package sqs

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"go-boilerplate/internal/observability"
	boundary "go-boilerplate/internal/usecase/boundary/publisher"
)

// AttrMessageID は、outbox の message_id を運ぶ MessageAttribute のキーです。
// SQS の MessageId は broker が採番する別物で、再 publish のたびに変わるため冪等キーには使えません。
// 受信側はこの属性を冪等キーの材料として読みます。
const AttrMessageID = "message_id"

// attrTypeString は、MessageAttribute の DataType です（SQS は型名の指定を必須とします）。
const attrTypeString = "String"

// egressHeaderDenylist は、broker へ送出してはならない機微ヘッダ名（小文字正規化済み）です。
// emit 側の denylist と重複しますが、emit を経由せず INSERT された行に対する egress 境界での防御です。
var egressHeaderDenylist = map[string]struct{}{
	"authorization":       {},
	"proxy-authorization": {},
	"cookie":              {},
	"set-cookie":          {},
}

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
// 送信失敗は apperror へ正規化して返し、再送は relay の次 poll が担います（at-least-once）。
func (p *publisher) Publish(ctx context.Context, m boundary.Message) error {
	ctx, endSpan := p.tracer.Start(ctx)
	defer endSpan()

	_, err := p.api.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:          aws.String(p.cfg.QueueURL),
		MessageBody:       aws.String(string(m.Payload)),
		MessageAttributes: p.messageAttributes(m),
	})
	return normalizeError(err)
}

// messageAttributes は、message_id と伝搬対象ヘッダを MessageAttributes へ組み立てます。
// 空値のヘッダは SQS が InvalidParameterValue で拒否するため落とします。
func (p *publisher) messageAttributes(m boundary.Message) map[string]types.MessageAttributeValue {
	attrs := make(map[string]types.MessageAttributeValue, len(m.Headers)+1)
	attrs[AttrMessageID] = types.MessageAttributeValue{
		DataType:    aws.String(attrTypeString),
		StringValue: aws.String(m.MessageID.String()),
	}

	for k, v := range m.Headers {
		if _, denied := egressHeaderDenylist[strings.ToLower(k)]; denied {
			continue
		}
		if k == AttrMessageID || v == "" {
			continue
		}
		attrs[k] = types.MessageAttributeValue{
			DataType:    aws.String(attrTypeString),
			StringValue: aws.String(v),
		}
	}
	return attrs
}
