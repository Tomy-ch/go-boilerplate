package sqs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/worker"
)

// 実装漏れをコンパイル時に検出します。
var _ worker.FailureHandler = (*DeadLetter)(nil)

// DeadLetter は、worker.FailureHandler の SQS 実装です（Permanent を別キュー=DLQ へ送ります）。
// SQS の redrive policy に委ねる運用ではこの実装を配線せず、app は ReceiveCount 監視のみとします。
type DeadLetter struct {
	api    API
	dlqURL string
	tracer observability.LayerTracer
}

// NewDeadLetter は、DLQ への退避を行う FailureHandler を生成します。
func NewDeadLetter(api API, dlqURL string, tf observability.TracerFactory) *DeadLetter {
	return &DeadLetter{api: api, dlqURL: dlqURL, tracer: tf.Infra()}
}

// Fail は、永久失敗メッセージを DLQ へ SendMessage します。失敗理由は属性に付与します。
func (d *DeadLetter) Fail(ctx context.Context, m worker.Message, cause error) error {
	ctx, endSpan := d.tracer.Start(ctx)
	defer endSpan()

	attrs := map[string]types.MessageAttributeValue{}
	if cause != nil {
		attrs["failure_reason"] = types.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(cause.Error()),
		}
	}

	_, err := d.api.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:          aws.String(d.dlqURL),
		MessageBody:       aws.String(string(m.Body)),
		MessageAttributes: attrs,
	})
	return normalizeError(err)
}
