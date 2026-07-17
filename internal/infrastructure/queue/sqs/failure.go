package sqs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/worker"
)

// attrDataTypeString は、SQS メッセージ属性の DataType に指定する文字列型の値です。
const attrDataTypeString = "String"

// 実装漏れをコンパイル時に検出します。
var _ worker.FailureHandler = (*deadLetter)(nil)

// deadLetter は、worker.FailureHandler の SQS 実装です（Permanent を別キュー=DLQ へ送ります）。
// SQS の redrive policy に委ねる運用ではこの実装を配線せず、app は ReceiveCount 監視のみとします。
type deadLetter struct {
	api    API
	dlqURL string
	tracer observability.LayerTracer
}

// NewDeadLetter は、DLQ への退避を行う FailureHandler を生成します。
func NewDeadLetter(api API, dlqURL string, tf observability.TracerFactory) worker.FailureHandler {
	return &deadLetter{api: api, dlqURL: dlqURL, tracer: tf.Infra()}
}

// Fail は、永久失敗メッセージを DLQ へ SendMessage します。
// 失敗理由は分類カテゴリのみ属性に付与し、cause の詳細は載せません
// （cause が PII/内部詳細を含みうるため。詳細は engine 側のログに残ります）。
func (d *deadLetter) Fail(ctx context.Context, m worker.Message, _ error) error {
	ctx, endSpan := d.tracer.Start(ctx)
	defer endSpan()

	_, err := d.api.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(d.dlqURL),
		MessageBody: aws.String(string(m.Body)),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"failure_reason": {
				DataType:    aws.String(attrDataTypeString),
				StringValue: aws.String("permanent"),
			},
		},
	})
	return normalizeError(err)
}
