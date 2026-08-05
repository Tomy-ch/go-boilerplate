package sqs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	boundary "go-boilerplate/internal/usecase/boundary/publisher"
	"go-boilerplate/internal/usecase/boundary/worker"
	"go-boilerplate/pkg/httpheader"
	"go-boilerplate/pkg/xerrors"
)

// AttrMessageID は、outbox の message_id を運ぶ MessageAttribute のキーです。
// SQS の MessageId は broker が採番する別物で、再 publish のたびに変わるため冪等キーには使えません。
// 受信側はこの属性を冪等キーの材料として読みます。
const AttrMessageID = "message_id"

// AttrEventType は、イベント種別を運ぶ MessageAttribute のキーです。
// 受信側 Handler が本文を parse する前に処理対象を選別できるよう、worker seam が定める属性名で載せます。
const AttrEventType = worker.AttrEventType

// attrTypeString は、MessageAttribute の DataType です（SQS は型名の指定を必須とします）。
const attrTypeString = "String"

// maxMessageAttributes は、SendMessage が受け付ける MessageAttributes の上限です（SQS の仕様）。
// 超過するとメッセージ全体が InvalidParameterValue で拒否されます。
const maxMessageAttributes = 10

// reservedAttributes は、adapter 自身が占める属性の数（message_id / event_type）です。
// 伝搬できるヘッダの数は、上限からこの分だけ減ります。
const reservedAttributes = 2

// ErrTooManyAttributes は、伝搬対象ヘッダが SQS の MessageAttributes 上限を超えたことを示すエラーです。
var ErrTooManyAttributes = xerrors.Wrap(apperror.ErrInvalidArgument, "too many message attributes")

// ErrMissingEventType は、イベント種別を持たないメッセージを publish しようとしたことを示すエラーです。
var ErrMissingEventType = xerrors.Wrap(apperror.ErrInvalidArgument, "missing event type")

// 実装漏れをコンパイル時に検出します。
var _ boundary.Publisher = (*publisher)(nil)

// PublisherConfig は、SQS publisher の adapter 固有設定です。
type PublisherConfig struct {
	// QueueURL は、publish 先キューの URL です。
	QueueURL string
}

// publisher は、boundary.Publisher の SQS 実装です。
type publisher struct {
	api    API
	cfg    PublisherConfig
	tracer observability.LayerTracer
}

// NewPublisher は、SQS publisher を生成します。
func NewPublisher(api API, cfg PublisherConfig, tf observability.TracerFactory) boundary.Publisher {
	return &publisher{api: api, cfg: cfg, tracer: tf.Infra()}
}

// Publish は、メッセージを SendMessage でキューへ送ります。
// outbox の message_id と伝搬対象ヘッダは MessageAttributes へ載せます。本文は payload そのままで、
// 受信側が本文を解釈せずに冪等キーを取り出せるようにするためです。
// 上限超過は送信前に ErrTooManyAttributes として返します。超過分を落とすと、どのヘッダが残るかが
// map の反復順に左右され、traceparent を失ったことにも気付けないためです。
// 送信失敗は apperror へ正規化して返し、再送は relay の次 poll が担います（at-least-once）。
func (p *publisher) Publish(ctx context.Context, m boundary.Message) error {
	ctx, endSpan := p.tracer.Start(ctx)
	defer endSpan()

	// SQS は空値の属性を拒むため、送れないメッセージは送信前に弾く。ヘッダと違って落として済ませられない。
	// 種別を欠いたメッセージは受信側が自分宛かを判定できず、黙って読み捨てられる。
	if m.EventType == "" {
		return xerrors.Wrap(ErrMissingEventType, "event type must not be empty")
	}

	attrs := p.messageAttributes(m)
	if len(attrs) > maxMessageAttributes {
		return xerrors.Wrap(ErrTooManyAttributes,
			fmt.Sprintf("%d attributes exceed the SQS limit of %d", len(attrs), maxMessageAttributes))
	}

	_, err := p.api.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:          aws.String(p.cfg.QueueURL),
		MessageBody:       aws.String(string(m.Payload)),
		MessageAttributes: attrs,
	})
	return normalizeError(err)
}

// messageAttributes は、message_id / event_type と伝搬対象ヘッダを MessageAttributes へ組み立てます。
// 空値のヘッダは SQS が InvalidParameterValue で拒否するため落とします。
func (p *publisher) messageAttributes(m boundary.Message) map[string]types.MessageAttributeValue {
	attrs := make(map[string]types.MessageAttributeValue, len(m.Headers)+reservedAttributes)
	attrs[AttrMessageID] = types.MessageAttributeValue{
		DataType:    aws.String(attrTypeString),
		StringValue: aws.String(m.MessageID.String()),
	}
	attrs[AttrEventType] = types.MessageAttributeValue{
		DataType:    aws.String(attrTypeString),
		StringValue: aws.String(m.EventType),
	}

	for k, v := range m.Headers {
		// emit 側でも同じ判定を行いますが、emit を経由せず INSERT された行に対する egress 境界での防御です。
		if httpheader.IsSensitive(k) {
			continue
		}
		// 同名ヘッダに outbox 由来の値を上書きさせません。受信側の選別が本文と食い違うためです。
		if k == AttrMessageID || k == AttrEventType || v == "" {
			continue
		}
		attrs[k] = types.MessageAttributeValue{
			DataType:    aws.String(attrTypeString),
			StringValue: aws.String(v),
		}
	}
	return attrs
}
