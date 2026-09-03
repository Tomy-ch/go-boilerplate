package aws

import (
	"context"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	publisherbndry "go-boilerplate/internal/usecase/boundary/publisher"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/xerrors"
)

// ErrEventIDMismatch は、payload の eventId が outbox の message_id と食い違うことを示すエラーです（分類は Publish の doc）。
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
	metrics  *observability.RealtimeMetrics
}

// NewPublisher は、log へ append し topicARN へ wakeup を publish する Publisher を返します。
func NewPublisher(
	log rt.EventLogStore,
	snsAPI SNSAPI,
	topicARN string,
	tf observability.TracerFactory,
	metrics *observability.RealtimeMetrics,
) publisherbndry.Publisher {
	return &publisher{log: log, sns: snsAPI, topicARN: topicARN, tracer: tf.Infra(), metrics: metrics}
}

// Publish は、m の payload を DeliveryEvent として復元し、EventLog へ append してから wakeup を publish します。
// 復元できない payload・eventId の不一致（ErrEventIDMismatch）・位置の衝突（ErrSequenceConflict）は retry で
// 直らないので ErrPermanent で返し、relay がその stream を先頭で止めます。EventLog と SNS の失敗は ErrRetryable
// です（分類表は README「Error classification」）。
func (p *publisher) Publish(ctx context.Context, m publisherbndry.Message) error {
	// 起点の command が headers に載せた trace を継いでから span を開きます。これをしないと
	// append が relay の trace にぶら下がり、command → outbox → relay → EventLog が 1 本になりません。
	ctx = observability.ExtractFromCarrier(ctx, m.Headers)

	ctx, endSpan := p.tracer.Start(ctx)
	defer endSpan()

	event, err := decodeEvent(m)
	if err != nil {
		return err
	}

	event.Origin = observability.TraceContextFromCarrier(m.Headers)

	if err := p.log.Append(ctx, event); err != nil {
		p.metrics.EventLogAppended(ctx, appendResult(err))

		return classifyAppend(err)
	}

	p.metrics.EventLogAppended(ctx, observability.RealtimeResultOK)

	// outbox に記録されてから EventLog に載るまでが、この経路の遅れです。
	// 起点を持たない封筒（relay 以外の呼び出し元）は、ゼロ値の差で histogram を壊すので数えません。
	if !m.CreatedAt.IsZero() {
		p.metrics.EventLogLag(ctx, float64(time.Since(m.CreatedAt).Milliseconds()))
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
		p.metrics.WakeupPublishFailed(ctx)

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

// appendResult は、append の失敗を realtime.eventlog.appends の result へ写します。
// 位置の衝突だけは「既に誰かが書いた」ことを表すので、substrate の失敗と分けて数えます。
func appendResult(err error) string {
	if xerrors.Is(err, rt.ErrSequenceConflict) {
		return observability.RealtimeResultConflict
	}

	return observability.RealtimeResultError
}

// classifyAppend は、append の失敗を relay の分類（Publish の doc）へ写します。
func classifyAppend(err error) error {
	if xerrors.Is(err, rt.ErrSequenceConflict) || xerrors.Is(err, rt.ErrInvalidEvent) ||
		xerrors.Is(err, rt.ErrPayloadTooLarge) {
		return xerrors.Join(apperror.ErrPermanent, err)
	}

	return xerrors.Join(apperror.ErrRetryable, err)
}
