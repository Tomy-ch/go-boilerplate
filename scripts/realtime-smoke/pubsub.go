package main

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"go-boilerplate/pkg/xerrors"
)

const (
	pubSubSubject = "GoAWS SNS/SQS"

	attrRawMessageDelivery = "RawMessageDelivery"
	attrEventType          = "eventType"
	eventTypeValue         = "smoke"

	// receiveWindow は、publish 後に各 queue で配送を待つ上限です。long polling を複数回繰り返します。
	receiveWindow = 20 * time.Second
	// receiveWait は、ReceiveMessage の WaitTimeSeconds です（production と同じ long polling）。
	receiveWait = 5
	// receiveBatch は、1 回の ReceiveMessage で受け取る最大件数です。重複配送を検出するため 1 より大きくします。
	receiveBatch = 10
)

// notification は、production が SNS に載せる本文と同じ形です（eventId / streamId / sequence のみ）。
type notification struct {
	EventID  string `json:"eventId"`
	StreamID string `json:"streamId"`
	Sequence string `json:"sequence"`
}

// queuePolicy は、対象 topic からの送信だけを許す SQS queue policy です。
type queuePolicy struct {
	Version   string            `json:"Version"`   //nolint:tagliatelle // IAM policy document のフィールド名
	Statement []policyStatement `json:"Statement"` //nolint:tagliatelle // 同上
}

type policyStatement struct {
	Sid       string            `json:"Sid"`       //nolint:tagliatelle // IAM policy document のフィールド名
	Effect    string            `json:"Effect"`    //nolint:tagliatelle // 同上
	Principal map[string]string `json:"Principal"` //nolint:tagliatelle // 同上
	Action    string            `json:"Action"`    //nolint:tagliatelle // 同上
	Resource  string            `json:"Resource"`  //nolint:tagliatelle // 同上
	Condition map[string]any    `json:"Condition"` //nolint:tagliatelle // 同上
}

// pubSubSmoke は、SNS / SQS 検査の状態です。
type pubSubSmoke struct {
	sns   *sns.Client
	sqs   *sqs.Client
	names names
	n     int

	topicArn  string
	queueURLs []string
	queueArns []string
	subArns   []string
	payload   string
	received  [][]sqstypes.Message
}

// runPubSub は、instance fan-out が依存する呼び出しを順に検査し、結果を記録します。
func runPubSub(ctx context.Context, snsClient *sns.Client, sqsClient *sqs.Client, n names, subscribers int, keep bool, rec *recorder) {
	s := &pubSubSmoke{sns: snsClient, sqs: sqsClient, names: n, n: subscribers}

	runChain(ctx, pubSubSubject, s.steps(), rec)
	s.cleanup(ctx, keep, rec)
}

func (s *pubSubSmoke) steps() []step {
	return []step{
		{id: "G0", check: "SQS wire protocol probe（ListQueues, AWS JSON 1.0）", halt: true, fn: s.probeProtocol},
		{id: "G1", check: "SNS CreateTopic", halt: true, fn: s.createTopic},
		{id: "G2", check: "SQS CreateQueue × N", halt: true, fn: s.createQueues},
		{id: "G3", check: "GetQueueAttributes QueueArn", halt: true, fn: s.queueArnsStep},
		{id: "G4", check: "SetQueueAttributes Policy（aws:SourceArn）→ 読み戻し", fn: s.queuePolicy},
		{id: "G4b", check: "CreateQueue Attributes.Policy → 読み戻し（G4 の代替経路）", fn: s.createQueueWithPolicy},
		{id: "G5", check: "Subscribe sqs + RawMessageDelivery=true → 読み戻し", halt: true, fn: s.subscribe},
		{id: "G6", check: "Publish 1 回 → N queue で受信、body が raw payload と一致", fn: s.publishAndReceive},
		{id: "G7", check: "MessageAttributes の透過（付随情報）", fn: s.messageAttributes},
		{id: "G8", check: "DeleteMessage（receipt handle）", fn: s.deleteMessages},
	}
}

func (s *pubSubSmoke) probeProtocol(ctx context.Context) (string, error) {
	if _, err := s.sqs.ListQueues(ctx, &sqs.ListQueuesInput{MaxResults: aws.Int32(1)}); err != nil {
		return "", err
	}

	return "SDK v2 の JSON protocol で ListQueues が応答した", nil
}

func (s *pubSubSmoke) createTopic(ctx context.Context) (string, error) {
	out, err := s.sns.CreateTopic(ctx, &sns.CreateTopicInput{Name: aws.String(s.names.topic())})
	if err != nil {
		return "", err
	}

	s.topicArn = aws.ToString(out.TopicArn)
	if s.topicArn == "" {
		return "", incompatible("TopicArn が空で返った")
	}

	return s.topicArn, nil
}

func (s *pubSubSmoke) createQueues(ctx context.Context) (string, error) {
	s.queueURLs = make([]string, 0, s.n)
	for i := range s.n {
		out, err := s.sqs.CreateQueue(ctx, &sqs.CreateQueueInput{QueueName: aws.String(s.names.queue(i))})
		if err != nil {
			return "", err
		}

		if aws.ToString(out.QueueUrl) == "" {
			return "", incompatible("QueueUrl が空で返った")
		}

		s.queueURLs = append(s.queueURLs, aws.ToString(out.QueueUrl))
	}

	return strconv.Itoa(len(s.queueURLs)) + " queue を作成（" + s.queueURLs[0] + " …）", nil
}

func (s *pubSubSmoke) queueAttribute(ctx context.Context, url string, name sqstypes.QueueAttributeName) (string, error) {
	out, err := s.sqs.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(url),
		AttributeNames: []sqstypes.QueueAttributeName{name},
	})
	if err != nil {
		return "", err
	}

	return out.Attributes[string(name)], nil
}

func (s *pubSubSmoke) queueArnsStep(ctx context.Context) (string, error) {
	s.queueArns = make([]string, 0, len(s.queueURLs))
	for _, url := range s.queueURLs {
		arn, err := s.queueAttribute(ctx, url, sqstypes.QueueAttributeNameQueueArn)
		if err != nil {
			return "", err
		}

		if arn == "" {
			return "", incompatible("QueueArn が空で返った: " + url)
		}

		s.queueArns = append(s.queueArns, arn)
	}

	return s.queueArns[0] + " …", nil
}

// policyDocument は、queueArn への送信を topic からだけ許す policy を返します。
func (s *pubSubSmoke) policyDocument(queueArn string) (string, error) {
	doc, err := json.Marshal(queuePolicy{
		Version: "2012-10-17",
		Statement: []policyStatement{{
			Sid:       "AllowTopic",
			Effect:    "Allow",
			Principal: map[string]string{"Service": "sns.amazonaws.com"},
			Action:    "sqs:SendMessage",
			Resource:  queueArn,
			Condition: map[string]any{"ArnEquals": map[string]string{"aws:SourceArn": s.topicArn}},
		}},
	})
	if err != nil {
		return "", xerrors.Wrap(err, "marshal policy")
	}

	return string(doc), nil
}

// verifyPolicy は、queue に保存された Policy が doc と同じ値かを読み戻して確かめます。
func (s *pubSubSmoke) verifyPolicy(ctx context.Context, url, doc string) error {
	got, err := s.queueAttribute(ctx, url, sqstypes.QueueAttributeNamePolicy)
	if err != nil {
		return err
	}

	if got == "" {
		return incompatible("Policy は受理されたが読み戻せない（silent drop）")
	}

	if !sameJSON(doc, got) {
		return incompatible("読み戻した Policy が設定値と異なる: " + got)
	}

	return nil
}

func (s *pubSubSmoke) queuePolicy(ctx context.Context) (string, error) {
	for i, url := range s.queueURLs {
		doc, err := s.policyDocument(s.queueArns[i])
		if err != nil {
			return "", err
		}

		if _, err := s.sqs.SetQueueAttributes(ctx, &sqs.SetQueueAttributesInput{
			QueueUrl:   aws.String(url),
			Attributes: map[string]string{string(sqstypes.QueueAttributeNamePolicy): doc},
		}); err != nil {
			return "", err
		}

		if err := s.verifyPolicy(ctx, url, doc); err != nil {
			return "", err
		}
	}

	return "aws:SourceArn 条件付き policy が保存され読み戻せた", nil
}

// createQueueWithPolicy は、作成時の Attributes で Policy を渡す経路を検査します。production の
// instance queue は作成と同時に policy を持つのが自然なので、SetQueueAttributes が通らない場合の代替経路になります。
func (s *pubSubSmoke) createQueueWithPolicy(ctx context.Context) (string, error) {
	name := s.names.queue(s.n)
	doc, err := s.policyDocument("arn:aws:sqs:" + defaultRegion + ":000000000000:" + name)
	if err != nil {
		return "", err
	}

	out, err := s.sqs.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName:  aws.String(name),
		Attributes: map[string]string{string(sqstypes.QueueAttributeNamePolicy): doc},
	})
	if err != nil {
		return "", err
	}

	url := aws.ToString(out.QueueUrl)
	verifyErr := s.verifyPolicy(ctx, url, doc)
	if _, err := s.sqs.DeleteQueue(ctx, &sqs.DeleteQueueInput{QueueUrl: aws.String(url)}); err != nil {
		return "", xerrors.Wrap(err, "delete queue "+url)
	}

	if verifyErr != nil {
		return "", verifyErr
	}

	return "作成時の Policy 属性が保存され読み戻せた", nil
}

// sameJSON は、整形差を無視して 2 つの JSON が同じ値かを判定します（object の key 順は正規化される）。
func sameJSON(a, b string) bool {
	ca, err := canonicalJSON(a)
	if err != nil {
		return false
	}

	cb, err := canonicalJSON(b)
	if err != nil {
		return false
	}

	return ca == cb
}

func canonicalJSON(s string) (string, error) {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return "", xerrors.Wrap(err, "unmarshal")
	}

	out, err := json.Marshal(v)
	if err != nil {
		return "", xerrors.Wrap(err, "marshal")
	}

	return string(out), nil
}

func (s *pubSubSmoke) subscribe(ctx context.Context) (string, error) {
	s.subArns = make([]string, 0, len(s.queueArns))
	for _, arn := range s.queueArns {
		out, err := s.sns.Subscribe(ctx, &sns.SubscribeInput{
			TopicArn:              aws.String(s.topicArn),
			Protocol:              aws.String("sqs"),
			Endpoint:              aws.String(arn),
			ReturnSubscriptionArn: true,
		})
		if err != nil {
			return "", err
		}

		subArn := aws.ToString(out.SubscriptionArn)
		if _, err := s.sns.SetSubscriptionAttributes(ctx, &sns.SetSubscriptionAttributesInput{
			SubscriptionArn: aws.String(subArn),
			AttributeName:   aws.String(attrRawMessageDelivery),
			AttributeValue:  aws.String("true"),
		}); err != nil {
			return "", err
		}

		got, err := s.sns.GetSubscriptionAttributes(ctx, &sns.GetSubscriptionAttributesInput{SubscriptionArn: aws.String(subArn)})
		if err != nil {
			return "", err
		}

		if got.Attributes[attrRawMessageDelivery] != "true" {
			return "", incompatible("RawMessageDelivery が読み戻せない: " + got.Attributes[attrRawMessageDelivery])
		}

		s.subArns = append(s.subArns, subArn)
	}

	return strconv.Itoa(len(s.subArns)) + " subscription で RawMessageDelivery=true が読み戻せた", nil
}

func (s *pubSubSmoke) publishAndReceive(ctx context.Context) (string, error) {
	body, err := json.Marshal(notification{EventID: "evt-" + s.names.runID, StreamID: streamID, Sequence: "1"})
	if err != nil {
		return "", xerrors.Wrap(err, "marshal notification")
	}

	s.payload = string(body)

	if _, err := s.sns.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(s.topicArn),
		Message:  aws.String(s.payload),
		MessageAttributes: map[string]snstypes.MessageAttributeValue{
			attrEventType: {DataType: aws.String("String"), StringValue: aws.String(eventTypeValue)},
		},
	}); err != nil {
		return "", err
	}

	s.received = make([][]sqstypes.Message, len(s.queueURLs))
	for i, url := range s.queueURLs {
		msgs, err := s.receive(ctx, url)
		if err != nil {
			return "", err
		}

		s.received[i] = msgs
	}

	return s.verifyDelivery()
}

// receive は、1 queue で配送を待ち、届いた message をすべて返します。
func (s *pubSubSmoke) receive(ctx context.Context, url string) ([]sqstypes.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, receiveWindow)
	defer cancel()

	for {
		out, err := s.sqs.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:              aws.String(url),
			MaxNumberOfMessages:   receiveBatch,
			WaitTimeSeconds:       receiveWait,
			MessageAttributeNames: []string{"All"},
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil, nil // 窓を使い切った。届かなかったことは verifyDelivery が非互換として扱う
			}

			return nil, err
		}

		if len(out.Messages) > 0 {
			return out.Messages, nil
		}

		if ctx.Err() != nil {
			return nil, nil
		}
	}
}

func (s *pubSubSmoke) verifyDelivery() (string, error) {
	delivered := 0
	for i, msgs := range s.received {
		switch {
		case len(msgs) == 0:
			continue
		case len(msgs) > 1:
			return "", incompatible("queue " + strconv.Itoa(i) + " に " + strconv.Itoa(len(msgs)) + " 件届いた（重複配送）")
		}

		body := aws.ToString(msgs[0].Body)
		if isSNSEnvelope(body) {
			return "", incompatible("RawMessageDelivery が効かず SNS envelope（Type: Notification）が届いた")
		}

		if body != s.payload {
			return "", incompatible("body が raw payload と一致しない: " + body)
		}

		delivered++
	}

	if delivered != len(s.received) {
		return "", incompatible(strconv.Itoa(len(s.received)) + " queue 中 " + strconv.Itoa(delivered) + " queue にしか届かない")
	}

	return strconv.Itoa(delivered) + " queue すべてに raw payload が 1 件ずつ届いた", nil
}

// isSNSEnvelope は、body が RawMessageDelivery 無効時の SNS envelope かを判定します。
func isSNSEnvelope(body string) bool {
	var env struct {
		Type string `json:"Type"` //nolint:tagliatelle // SNS envelope のフィールド名
	}

	return json.Unmarshal([]byte(body), &env) == nil && env.Type == "Notification"
}

func (s *pubSubSmoke) messageAttributes(_ context.Context) (string, error) {
	if s.received == nil {
		return "", incompatible("先行検査 G6 で message を受け取れていない")
	}

	for i, msgs := range s.received {
		if len(msgs) == 0 {
			return "", incompatible("queue " + strconv.Itoa(i) + " に message が無い")
		}

		attr, ok := msgs[0].MessageAttributes[attrEventType]
		if !ok {
			return "", incompatible("MessageAttributes が raw delivery で落ちる")
		}

		if aws.ToString(attr.StringValue) != eventTypeValue {
			return "", incompatible("MessageAttributes の値が変わる: " + aws.ToString(attr.StringValue))
		}
	}

	return "String 属性が raw delivery でも透過した", nil
}

func (s *pubSubSmoke) deleteMessages(ctx context.Context) (string, error) {
	if s.received == nil {
		return "", incompatible("先行検査 G6 で message を受け取れていない")
	}

	deleted := 0
	for i, msgs := range s.received {
		for _, m := range msgs {
			if _, err := s.sqs.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(s.queueURLs[i]),
				ReceiptHandle: m.ReceiptHandle,
			}); err != nil {
				return "", err
			}

			deleted++
		}
	}

	return strconv.Itoa(deleted) + " 件を削除", nil
}

// cleanup は、unsubscribe → queue 削除 → topic 削除 の順（orphan cleanup と同じ順）で後片付けします。
// 作られなかった resource は飛ばし、失敗は行として残します。
func (s *pubSubSmoke) cleanup(ctx context.Context, keep bool, rec *recorder) {
	const id, check = "G9", "Unsubscribe → DeleteQueue → DeleteTopic（後片付け）"

	switch {
	case s.topicArn == "" && len(s.queueURLs) == 0:
		rec.skip(id, pubSubSubject, check, "先行検査により作成された resource が無い")
	case keep:
		rec.skip(id, pubSubSubject, check, "-keep により未実施（topic "+s.topicArn+" が残っている）")
	default:
		rec.record(id, pubSubSubject, check, "subscription / queue / topic を削除", s.teardown(ctx))
	}
}

func (s *pubSubSmoke) teardown(ctx context.Context) error {
	for _, arn := range s.subArns {
		if _, err := s.sns.Unsubscribe(ctx, &sns.UnsubscribeInput{SubscriptionArn: aws.String(arn)}); err != nil {
			return xerrors.Wrap(err, "unsubscribe "+arn)
		}
	}

	for _, url := range s.queueURLs {
		if _, err := s.sqs.DeleteQueue(ctx, &sqs.DeleteQueueInput{QueueUrl: aws.String(url)}); err != nil {
			return xerrors.Wrap(err, "delete queue "+url)
		}
	}

	if s.topicArn != "" {
		if _, err := s.sns.DeleteTopic(ctx, &sns.DeleteTopicInput{TopicArn: aws.String(s.topicArn)}); err != nil {
			return xerrors.Wrap(err, "delete topic "+s.topicArn)
		}
	}

	return nil
}
