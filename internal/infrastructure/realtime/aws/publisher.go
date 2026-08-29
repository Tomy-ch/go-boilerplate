package aws

import (
	"context"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	publisherbndry "go-boilerplate/internal/usecase/boundary/publisher"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/xerrors"
)

// ErrEventIDMismatch は、payload の eventId が outbox の message_id と食い違うことを示すエラーです。
// eventId は message_id と同じ値で冪等性を決めるので、食い違いは emit 側の誤りであり retry では直りません。
var ErrEventIDMismatch = xerrors.Wrap(
	apperror.ErrPermanent,
	"realtime: payload eventId does not match the outbox message id",
)

var _ publisherbndry.Publisher = (*publisher)(nil)

// publisher は、realtime channel の outbox publisher です。EventLog へ append してから wakeup を publish します。
type publisher struct {
	log      rt.EventLogStore
	sns      SNSAPI
	topicARN string
	tracer   observability.LayerTracer
}

// NewPublisher は、log へ append し topicARN へ wakeup を publish する Publisher を返します。
// append は同じ EventID なら冪等に成功するので、再送で EventLog に 2 件目は入りません（ADR-0073）。
func NewPublisher(
	log rt.EventLogStore,
	snsAPI SNSAPI,
	topicARN string,
	tf observability.TracerFactory,
) publisherbndry.Publisher {
	return &publisher{log: log, sns: snsAPI, topicARN: topicARN, tracer: tf.Infra()}
}

// Publish は、m の payload を DeliveryEvent として復元し、EventLog へ append してから wakeup を publish します。
// 復元できない payload と、同じ位置に別の event がある衝突（ErrSequenceConflict）は retry で直らないので
// ErrPermanent で返し、relay がその stream を先頭で止めます。substrate の失敗は ErrRetryable です。
func (p *publisher) Publish(ctx context.Context, m publisherbndry.Message) error {
	ctx, endSpan := p.tracer.Start(ctx)
	defer endSpan()

	event, err := decodeEvent(m)
	if err != nil {
		return err
	}

	if err := p.log.Append(ctx, event); err != nil {
		return classifyAppend(err)
	}

	body, attrs, err := encodeWakeup(
		rt.Wakeup{EventID: event.EventID, StreamID: event.StreamID, Sequence: event.Sequence},
	)
	if err != nil {
		return xerrors.Join(apperror.ErrPermanent, err)
	}

	if _, err := p.sns.Publish(ctx, &sns.PublishInput{
		TopicArn:          awssdk.String(p.topicARN),
		Message:           awssdk.String(body),
		MessageAttributes: attrs,
	}); err != nil {
		return xerrors.Join(apperror.ErrRetryable, normalize(err, "publish wakeup"))
	}

	return nil
}

// decodeEvent は、payload を DeliveryEvent へ復元します。eventId が空なら outbox の message_id で埋め、
// 入っていて食い違えば ErrEventIDMismatch です。
func decodeEvent(m publisherbndry.Message) (rt.DeliveryEvent, error) {
	event, err := rt.ParseDeliveryEvent(m.Payload)
	if err != nil {
		return rt.DeliveryEvent{}, xerrors.Join(apperror.ErrPermanent, err)
	}

	messageID := m.MessageID.String()
	switch event.EventID {
	case "":
		event.EventID = messageID
	case messageID:
	default:
		return rt.DeliveryEvent{}, xerrors.Wrap(ErrEventIDMismatch, event.EventID+" != "+messageID)
	}

	if err := event.Validate(); err != nil {
		return rt.DeliveryEvent{}, xerrors.Join(apperror.ErrPermanent, err)
	}

	return event, nil
}

// classifyAppend は、append の失敗を relay の分類へ写します。位置の衝突と不正な封筒は permanent、
// それ以外（store に届かない・書けない）は retryable です。
func classifyAppend(err error) error {
	if xerrors.Is(err, rt.ErrSequenceConflict) || xerrors.Is(err, rt.ErrInvalidEvent) ||
		xerrors.Is(err, rt.ErrPayloadTooLarge) {
		return xerrors.Join(apperror.ErrPermanent, err)
	}

	return xerrors.Join(apperror.ErrRetryable, err)
}
