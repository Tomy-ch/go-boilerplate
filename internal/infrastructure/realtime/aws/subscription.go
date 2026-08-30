package aws

import (
	"context"
	"regexp"
	"sync"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/xerrors"
)

const (
	// maxReceiveBatch は、ReceiveMessage の最大取得件数（SQS の仕様上限）です。
	maxReceiveBatch = 10
	// maxQueueNameLen は、SQS の queue 名の上限です。
	maxQueueNameLen = 80
	// subscriptionProtocol は、SNS から SQS へ届ける subscription の protocol です。
	subscriptionProtocol = "sqs"
	// attrRawMessageDelivery は、SNS envelope を剥がして本文だけを届けさせる subscription 属性です。
	attrRawMessageDelivery = "RawMessageDelivery"
)

// ErrInvalidQueueName は、prefix と instance の識別子から作った queue 名が SQS の制約を満たさないことを示すエラーです。
var ErrInvalidQueueName = xerrors.Wrap(apperror.ErrInvalidArgument, "realtime: invalid instance queue name")

// queueNameRe は、SQS の queue 名に使える文字です（standard queue）。
var queueNameRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

var _ rt.InstanceSubscription = (*subscription)(nil)

// SubscriptionTarget は、instance の受信先をどの topic に、どの名前で作るかです。
type SubscriptionTarget struct {
	// TopicARN は、subscribe する topic です。
	TopicARN string
	// QueuePrefix は、instance queue 名の先頭です（名前は <QueuePrefix>-<instance id>）。
	QueuePrefix string
}

// subscription は、rt.InstanceSubscription の SNS / SQS 実装です。1 つの instance が 1 組の queue と
// subscription を持ち、その生存期間に閉じます。
type subscription struct {
	sns    SNSAPI
	sqs    SQSAPI
	target SubscriptionTarget
	attrs  AttributesBuilder
	tracer observability.LayerTracer

	mu              sync.Mutex
	instanceID      rt.InstanceID
	queueURL        string
	queueARN        string
	subscriptionARN string
}

// NewInstanceSubscription は、target の topic に対する instance 固有の受信先を管理する InstanceSubscription を返します。
func NewInstanceSubscription(
	snsAPI SNSAPI, sqsAPI SQSAPI, target SubscriptionTarget, attrs AttributesBuilder, tf observability.TracerFactory,
) rt.InstanceSubscription {
	return &subscription{sns: snsAPI, sqs: sqsAPI, target: target, attrs: attrs, tracer: tf.Infra()}
}

// QueueName は、prefix と instance の識別子から instance queue の名前を返します。SQS の制約（使える文字と長さ）を
// 満たさなければ ErrInvalidQueueName を返します。
func QueueName(prefix string, id rt.InstanceID) (string, error) {
	name := prefix + "-" + string(id)
	if len(name) > maxQueueNameLen || !queueNameRe.MatchString(name) {
		return "", xerrors.Wrap(ErrInvalidQueueName, name)
	}

	return name, nil
}

// Provision は、queue を作り、属性を設定し、topic へ subscribe して RawMessageDelivery を有効にします。
// 途中で失敗したら作った分を片付けてから返します（起動失敗の instance が resource を残さないため）。
func (s *subscription) Provision(ctx context.Context, id rt.InstanceID) error {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.subscriptionARN != "" {
		if s.instanceID == id {
			return nil
		}

		return xerrors.Wrap(apperror.ErrConflict, "realtime: already provisioned for another instance")
	}

	name, err := QueueName(s.target.QueuePrefix, id)
	if err != nil {
		return err
	}

	if err := s.provision(ctx, name); err != nil {
		// 片付けの失敗も返す。握り潰すと「resource は残っていない」と「消せなかった」を呼び出し側が区別できない。
		return xerrors.Join(err, s.teardown(ctx))
	}

	s.instanceID = id

	return nil
}

// Receive は、long polling で最大 limit 件（0 以下と 10 超は SQS の上限 10 件）を受け取り、Notification へ復元して返します。
func (s *subscription) Receive(ctx context.Context, limit int) ([]rt.Notification, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	queueURL, err := s.currentQueueURL()
	if err != nil {
		return nil, err
	}

	if limit <= 0 || limit > maxReceiveBatch {
		limit = maxReceiveBatch
	}

	out, err := s.sqs.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:              awssdk.String(queueURL),
		MaxNumberOfMessages:   int32(limit),
		WaitTimeSeconds:       receiveWaitSeconds,
		MessageAttributeNames: []string{"All"},
	})
	if err != nil {
		return nil, s.classifyReceivingEnd(err, "receive notifications")
	}

	notifications := make([]rt.Notification, 0, len(out.Messages))
	for _, m := range out.Messages {
		notifications = append(notifications, decodeNotification(m))
	}

	return notifications, nil
}

// Delete は、処理済みの通知を queue から取り除きます。
func (s *subscription) Delete(ctx context.Context, n rt.Notification) error {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	queueURL, err := s.currentQueueURL()
	if err != nil {
		return err
	}

	if _, err := s.sqs.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl: awssdk.String(queueURL), ReceiptHandle: awssdk.String(n.Receipt),
	}); err != nil {
		return s.classifyReceivingEnd(err, "delete notification")
	}

	return nil
}

// Teardown は、unsubscribe → queue 削除の順に片付けます。片方が失敗しても残りを試み、失敗をまとめて返します
// （残った resource は orphan cleanup が lease から辿って回収する）。Provision していなければ何もしません。
func (s *subscription) Teardown(ctx context.Context) error {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.teardown(ctx)
}

// classifyReceivingEnd は、受信の失敗を分類します。受信先がもう無いなら、キャッシュした識別子を捨てて
// ErrReceivingEndGone を返します。作り直しはここでは行いません — 順序を持つ呼び出し側の役割です
// （package README の Port mapping / docs/design/realtime-delivery.md §2.5）。
func (s *subscription) classifyReceivingEnd(cause error, op string) error {
	if !queueGone(cause) {
		return normalize(cause, op)
	}

	s.invalidate()

	return xerrors.Join(rt.ErrReceivingEndGone, normalize(cause, op))
}

// invalidate は、キャッシュした受信先の識別子を捨てます。Provision は subscription の ARN が残っていると
// 早期に返るため、作り直しを受け付ける状態へ戻すにはこれが要ります。捨てるのは受信先の識別子だけで、
// どの instance のものかという帰属は失われていません。
func (s *subscription) invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.queueURL, s.queueARN, s.subscriptionARN = "", "", ""
}

func (s *subscription) provision(ctx context.Context, name string) error {
	created, err := s.sqs.CreateQueue(ctx, &sqs.CreateQueueInput{QueueName: awssdk.String(name)})
	if err != nil {
		return normalize(err, "create instance queue")
	}

	s.queueURL = awssdk.ToString(created.QueueUrl)

	got, err := s.sqs.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       awssdk.String(s.queueURL),
		AttributeNames: []sqstypes.QueueAttributeName{sqstypes.QueueAttributeNameQueueArn},
	})
	if err != nil {
		return normalize(err, "resolve instance queue arn")
	}

	s.queueARN = got.Attributes[string(sqstypes.QueueAttributeNameQueueArn)]
	if s.queueARN == "" {
		return xerrors.Wrap(apperror.ErrUnavailable, "resolve instance queue arn: empty QueueArn")
	}

	attrs, err := s.attrs.Build(s.queueARN)
	if err != nil {
		return err
	}

	if len(attrs) > 0 {
		if _, err := s.sqs.SetQueueAttributes(ctx, &sqs.SetQueueAttributesInput{
			QueueUrl: awssdk.String(s.queueURL), Attributes: attrs,
		}); err != nil {
			return normalize(err, "set instance queue attributes")
		}
	}

	sub, err := s.sns.Subscribe(ctx, &sns.SubscribeInput{
		TopicArn:              awssdk.String(s.target.TopicARN),
		Protocol:              awssdk.String(subscriptionProtocol),
		Endpoint:              awssdk.String(s.queueARN),
		ReturnSubscriptionArn: true,
	})
	if err != nil {
		return normalize(err, "subscribe instance queue")
	}

	s.subscriptionARN = awssdk.ToString(sub.SubscriptionArn)
	if s.subscriptionARN == "" {
		return xerrors.Wrap(apperror.ErrUnavailable, "subscribe instance queue: empty SubscriptionArn")
	}

	if _, err := s.sns.SetSubscriptionAttributes(ctx, &sns.SetSubscriptionAttributesInput{
		SubscriptionArn: awssdk.String(s.subscriptionARN),
		AttributeName:   awssdk.String(attrRawMessageDelivery),
		AttributeValue:  awssdk.String("true"),
	}); err != nil {
		return normalize(err, "enable raw message delivery")
	}

	return nil
}

func (s *subscription) teardown(ctx context.Context) error {
	var errs []error

	if s.subscriptionARN != "" {
		if _, err := s.sns.Unsubscribe(ctx, &sns.UnsubscribeInput{SubscriptionArn: awssdk.String(s.subscriptionARN)}); err != nil {
			errs = append(errs, normalize(err, "unsubscribe instance queue"))
		} else {
			s.subscriptionARN = ""
		}
	}

	if s.queueURL != "" {
		if _, err := s.sqs.DeleteQueue(ctx, &sqs.DeleteQueueInput{QueueUrl: awssdk.String(s.queueURL)}); err != nil {
			errs = append(errs, normalize(err, "delete instance queue"))
		} else {
			s.queueURL, s.queueARN = "", ""
		}
	}

	if len(errs) == 0 {
		s.instanceID = ""
	}

	return xerrors.Join(errs...)
}

// currentQueueURL は、Provision 済みの queue URL を返します。未 provision なら ErrReceivingEndGone です。
func (s *subscription) currentQueueURL() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.queueURL == "" {
		// 未 provision も同じ sentinel に畳む — 分けると復旧経路が割れる（README の Port mapping）。
		return "", rt.ErrReceivingEndGone
	}

	return s.queueURL, nil
}
