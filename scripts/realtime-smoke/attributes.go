package main

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

	realtimeaws "go-boilerplate/internal/infrastructure/realtime/aws"
	"go-boilerplate/pkg/xerrors"
)

const (
	// attrNotificationType は、wakeup と失効通知を 1 つの topic で見分けるための MessageAttribute 名です。
	// production と同じ定数を参照し、production 側の改名で smoke が旧名を検査し続けないようにします。
	attrNotificationType = realtimeaws.AttrKind
	typeWakeup           = "wakeup"
	typeRevocation       = "revocation"

	// 以下は production が instance queue に設定する値と同じです。emulator が受理し読み戻せるかを見ます。
	queueVisibilityTimeout = "30"
	queueReceiveWait       = "20"
	redriveMaxReceiveCount = "5"
	kmsKeyAlias            = "alias/aws/sqs"
)

// redrivePolicy は、SQS の RedrivePolicy 属性の JSON です。
type redrivePolicy struct {
	DeadLetterTargetArn string `json:"deadLetterTargetArn"`
	MaxReceiveCount     string `json:"maxReceiveCount"`
}

// revocation は、失効通知の本文です（subject / destination のみ）。
type revocation struct {
	Subject     string `json:"subject"`
	Destination string `json:"destination"`
}

// attributeSteps は、通知種別の振り分け、subscription の一覧、instance queue の属性を検査します。
// 属性を変えた後の受信（G13 / G15）は、属性が配送の振る舞いを変えないことを見るため、属性設定の直後に置きます。
// いずれも fan-out の成否とは独立なので halt は付けません。
func (s *pubSubSmoke) attributeSteps() []step {
	return []step{
		{id: "G10", check: "MessageAttribute type で wakeup / revocation を振り分け", fn: s.notificationTypes},
		{id: "G11", check: "ListSubscriptionsByTopic で queue ARN から subscription を引く", fn: s.listSubscriptions},
		{id: "G12", check: "SetQueueAttributes VisibilityTimeout / ReceiveMessageWaitTimeSeconds → 読み戻し", fn: s.queueTimings},
		{id: "G13", check: "属性設定後の Publish → 受信で重複配送が起きない", fn: s.receiveAfterAttributes},
		{id: "G14", check: "SetQueueAttributes RedrivePolicy（別 queue を DLQ に指定）→ 読み戻し", fn: s.redrive},
		{id: "G15", check: "RedrivePolicy 設定後の Publish → 受信で重複配送が起きない", fn: s.receiveAfterAttributes},
		{id: "G16", check: "SetQueueAttributes SqsManagedSseEnabled=true → 読み戻し", fn: s.managedSSE},
		{id: "G17", check: "SetQueueAttributes KmsMasterKeyId → 読み戻し", fn: s.kmsKey},
	}
}

// setAndReadBack は、queue に attrs を設定し、1 つずつ読み戻して同じ値かを確かめます。
// 受理されたのに読み戻せない属性は silent drop として非互換にします。
func (s *pubSubSmoke) setAndReadBack(ctx context.Context, url string, attrs map[sqstypes.QueueAttributeName]string) error {
	set := make(map[string]string, len(attrs))
	for name, value := range attrs {
		set[string(name)] = value
	}

	if _, err := s.sqs.SetQueueAttributes(ctx, &sqs.SetQueueAttributesInput{QueueUrl: aws.String(url), Attributes: set}); err != nil {
		return err
	}

	for name, want := range attrs {
		got, err := s.queueAttribute(ctx, url, name)
		if err != nil {
			return err
		}

		if got == "" {
			return incompatible(string(name) + " は受理されたが読み戻せない（silent drop）")
		}

		if got != want && !sameJSON(want, got) {
			return incompatible("読み戻した " + string(name) + " が設定値と異なる: " + got)
		}
	}

	return nil
}

func (s *pubSubSmoke) firstQueue() (string, error) {
	if len(s.queueURLs) == 0 {
		return "", incompatible("先行検査 G2 で queue を作れていない")
	}

	return s.queueURLs[0], nil
}

func (s *pubSubSmoke) queueTimings(ctx context.Context) (string, error) {
	url, err := s.firstQueue()
	if err != nil {
		return "", err
	}

	if err := s.setAndReadBack(ctx, url, map[sqstypes.QueueAttributeName]string{
		sqstypes.QueueAttributeNameVisibilityTimeout:             queueVisibilityTimeout,
		sqstypes.QueueAttributeNameReceiveMessageWaitTimeSeconds: queueReceiveWait,
	}); err != nil {
		return "", err
	}

	return "VisibilityTimeout=" + queueVisibilityTimeout + " / ReceiveMessageWaitTimeSeconds=" + queueReceiveWait + " が読み戻せた", nil
}

// redrive は、DLQ 用の queue を別に作り、先頭 queue の RedrivePolicy にその ARN を指定します。
// DLQ は後片付け（G9）で削除します。
func (s *pubSubSmoke) redrive(ctx context.Context) (string, error) {
	url, err := s.firstQueue()
	if err != nil {
		return "", err
	}

	out, err := s.sqs.CreateQueue(ctx, &sqs.CreateQueueInput{QueueName: aws.String(s.names.dlq())})
	if err != nil {
		return "", err
	}

	s.dlqURL = aws.ToString(out.QueueUrl)

	dlqArn, err := s.queueAttribute(ctx, s.dlqURL, sqstypes.QueueAttributeNameQueueArn)
	if err != nil {
		return "", err
	}

	if dlqArn == "" {
		return "", incompatible("DLQ の QueueArn が空で返った")
	}

	doc, err := json.Marshal(redrivePolicy{DeadLetterTargetArn: dlqArn, MaxReceiveCount: redriveMaxReceiveCount})
	if err != nil {
		return "", xerrors.Wrap(err, "marshal redrive policy")
	}

	if err := s.setAndReadBack(ctx, url, map[sqstypes.QueueAttributeName]string{
		sqstypes.QueueAttributeNameRedrivePolicy: string(doc),
	}); err != nil {
		return "", err
	}

	return "RedrivePolicy（maxReceiveCount=" + redriveMaxReceiveCount + "）が保存され読み戻せた", nil
}

func (s *pubSubSmoke) managedSSE(ctx context.Context) (string, error) {
	url, err := s.firstQueue()
	if err != nil {
		return "", err
	}

	if err := s.setAndReadBack(ctx, url, map[sqstypes.QueueAttributeName]string{
		sqstypes.QueueAttributeNameSqsManagedSseEnabled: "true",
	}); err != nil {
		return "", err
	}

	return "SqsManagedSseEnabled=true が読み戻せた", nil
}

func (s *pubSubSmoke) kmsKey(ctx context.Context) (string, error) {
	url, err := s.firstQueue()
	if err != nil {
		return "", err
	}

	if err := s.setAndReadBack(ctx, url, map[sqstypes.QueueAttributeName]string{
		sqstypes.QueueAttributeNameKmsMasterKeyId: kmsKeyAlias,
	}); err != nil {
		return "", err
	}

	return "KmsMasterKeyId=" + kmsKeyAlias + " が読み戻せた", nil
}

// listSubscriptions は、topic の subscription 一覧から先頭 queue の ARN を endpoint に持つものを引き、
// Subscribe が返した ARN と一致することを確かめます（orphan cleanup が lease だけから resource を辿る前提）。
func (s *pubSubSmoke) listSubscriptions(ctx context.Context) (string, error) {
	if len(s.queueArns) == 0 || len(s.subArns) == 0 {
		return "", incompatible("先行検査 G3 / G5 で queue ARN と subscription を得られていない")
	}

	var found *snstypes.Subscription

	p := sns.NewListSubscriptionsByTopicPaginator(s.sns, &sns.ListSubscriptionsByTopicInput{TopicArn: aws.String(s.topicArn)})
	for p.HasMorePages() && found == nil {
		page, err := p.NextPage(ctx)
		if err != nil {
			return "", err
		}

		for i := range page.Subscriptions {
			if aws.ToString(page.Subscriptions[i].Endpoint) == s.queueArns[0] {
				found = &page.Subscriptions[i]

				break
			}
		}
	}

	if found == nil {
		return "", incompatible("queue ARN " + s.queueArns[0] + " を endpoint に持つ subscription が一覧に無い")
	}

	if aws.ToString(found.SubscriptionArn) != s.subArns[0] {
		return "", incompatible("一覧の SubscriptionArn が Subscribe の戻り値と異なる: " + aws.ToString(found.SubscriptionArn))
	}

	return "queue ARN から subscription を引けた（" + s.subArns[0] + "）", nil
}

// notificationTypes は、wakeup と失効通知を type 属性だけ変えて publish し、受信側で属性から振り分けられることを確かめます。
func (s *pubSubSmoke) notificationTypes(ctx context.Context) (string, error) {
	url, err := s.firstQueue()
	if err != nil {
		return "", err
	}

	wakeup, err := json.Marshal(notification{EventID: "evt-" + s.names.runID + "-2", StreamID: streamID, Sequence: "2"})
	if err != nil {
		return "", xerrors.Wrap(err, "marshal wakeup")
	}

	revoked, err := json.Marshal(revocation{Subject: "subject-" + s.names.runID, Destination: streamID})
	if err != nil {
		return "", xerrors.Wrap(err, "marshal revocation")
	}

	for kind, body := range map[string][]byte{typeWakeup: wakeup, typeRevocation: revoked} {
		if _, err := s.sns.Publish(ctx, &sns.PublishInput{
			TopicArn: aws.String(s.topicArn),
			Message:  aws.String(string(body)),
			MessageAttributes: map[string]snstypes.MessageAttributeValue{
				attrNotificationType: {DataType: aws.String("String"), StringValue: aws.String(kind)},
			},
		}); err != nil {
			return "", err
		}
	}

	msgs, err := s.receive(ctx, url)
	if err != nil {
		return "", err
	}

	seen, err := dispatchByType(msgs)
	if err != nil {
		return "", err
	}

	for _, m := range msgs {
		if _, err := s.sqs.DeleteMessage(ctx, &sqs.DeleteMessageInput{QueueUrl: aws.String(url), ReceiptHandle: m.ReceiptHandle}); err != nil {
			return "", err
		}
	}

	return strconv.Itoa(len(msgs)) + " 件を type 属性で振り分けた（" + seen + "）", nil
}

// receiveAfterAttributes は、queue 属性（VisibilityTimeout / ReceiveMessageWaitTimeSeconds）を設定した後に
// 1 件 publish し、先頭 queue で 1 件だけ届くことを確かめます。long polling を queue 属性で有効にした
// emulator が同じ message を繰り返し返すと、consumer 側の重複が queue 属性の有無で変わるためです。
func (s *pubSubSmoke) receiveAfterAttributes(ctx context.Context) (string, error) {
	url, err := s.firstQueue()
	if err != nil {
		return "", err
	}

	body, err := json.Marshal(notification{EventID: "evt-" + s.names.runID + "-3", StreamID: streamID, Sequence: "3"})
	if err != nil {
		return "", xerrors.Wrap(err, "marshal wakeup")
	}

	if _, err := s.sns.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(s.topicArn),
		Message:  aws.String(string(body)),
		MessageAttributes: map[string]snstypes.MessageAttributeValue{
			attrNotificationType: {DataType: aws.String("String"), StringValue: aws.String(typeWakeup)},
		},
	}); err != nil {
		return "", err
	}

	msgs, err := s.receive(ctx, url)
	if err != nil {
		return "", err
	}

	// 属性設定後に受信と削除の対応が崩れる emulator があるため、削除の失敗は transport ではなく事後条件の不成立として扱う。
	for _, m := range msgs {
		if _, err := s.sqs.DeleteMessage(ctx, &sqs.DeleteMessageInput{QueueUrl: aws.String(url), ReceiptHandle: m.ReceiptHandle}); err != nil {
			return "", incompatible("受信した message を削除できない: " + err.Error())
		}
	}

	switch len(msgs) {
	case 1:
	case 0:
		return "", incompatible("1 件 publish して 1 件も届かない（属性設定後の配送停止）")
	default:
		return "", incompatible("1 件 publish して " + strconv.Itoa(len(msgs)) + " 件届いた（属性設定後の重複配送）")
	}

	return "属性設定後も 1 件だけ届いた", nil
}

// dispatchByType は、受信した message を type 属性で振り分け、wakeup と revocation が 1 件ずつ揃っているかを確かめます。
func dispatchByType(msgs []sqstypes.Message) (string, error) {
	counts := map[string]int{}
	for _, m := range msgs {
		attr, ok := m.MessageAttributes[attrNotificationType]
		if !ok {
			return "", incompatible("type 属性の無い message が届いた: " + aws.ToString(m.Body))
		}

		kind := aws.ToString(attr.StringValue)
		switch kind {
		case typeWakeup:
			if !isNotification(aws.ToString(m.Body)) {
				return "", incompatible("wakeup の body が通知の形でない: " + aws.ToString(m.Body))
			}
		case typeRevocation:
		default:
			return "", incompatible("未知の type 属性: " + kind)
		}

		counts[kind]++
	}

	if counts[typeWakeup] != 1 || counts[typeRevocation] != 1 {
		return "", incompatible("wakeup " + strconv.Itoa(counts[typeWakeup]) + " 件 / revocation " +
			strconv.Itoa(counts[typeRevocation]) + " 件（期待は 1 件ずつ）")
	}

	return typeWakeup + " 1 / " + typeRevocation + " 1", nil
}

// isNotification は、body が eventId / streamId / sequence を持つ通知かを判定します。
func isNotification(body string) bool {
	var n notification

	return json.Unmarshal([]byte(body), &n) == nil && n.EventID != "" && n.StreamID != "" && n.Sequence != ""
}
