//go:generate mockgen -source=$GOFILE -destination=mock/mock_sqs.gen.go -package=mock_$GOPACKAGE

package sqs

import (
	"context"
	"math"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/worker"
)

// maxSQSBatch は、ReceiveMessage の最大取得件数（SQS の仕様上限）です。
const maxSQSBatch = 10

// 実装漏れをコンパイル時に検出します。
var _ worker.Consumer = (*consumer)(nil)

// API は、Consumer が利用する SQS の操作のみを抽象化したものです（*sqs.Client が満たします）。
type API interface {
	ReceiveMessage(ctx context.Context, in *sqs.ReceiveMessageInput, opts ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, in *sqs.DeleteMessageInput, opts ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
	ChangeMessageVisibility(
		ctx context.Context,
		in *sqs.ChangeMessageVisibilityInput,
		opts ...func(*sqs.Options),
	) (*sqs.ChangeMessageVisibilityOutput, error)
	SendMessage(ctx context.Context, in *sqs.SendMessageInput, opts ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

// consumer は、worker.Consumer の SQS 実装です。
type consumer struct {
	api    API
	cfg    Config
	tracer observability.LayerTracer
}

// NewConsumer は、SQS Consumer を生成します。
func NewConsumer(api API, cfg Config, tf observability.TracerFactory) worker.Consumer {
	return &consumer{api: api, cfg: cfg, tracer: tf.Infra()}
}

// Receive は、ReceiveMessage で long-poll し、broker 非依存の Message へ変換して返します。
func (c *consumer) Receive(ctx context.Context, maxMessages int) ([]worker.Message, error) {
	ctx, endSpan := c.tracer.Start(ctx)
	defer endSpan()

	limit := maxMessages
	if c.cfg.MaxMessages > 0 && limit > int(c.cfg.MaxMessages) {
		limit = int(c.cfg.MaxMessages)
	}
	if limit > maxSQSBatch {
		limit = maxSQSBatch
	}
	if limit < 1 {
		limit = 1
	}
	n := int32(limit) // limit は 1..maxSQSBatch(10) に丸め済み

	out, err := c.api.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(c.cfg.QueueURL),
		MaxNumberOfMessages: n,
		WaitTimeSeconds:     c.cfg.WaitTimeSeconds,
		VisibilityTimeout:   c.cfg.VisibilityTimeout,
		MessageSystemAttributeNames: []types.MessageSystemAttributeName{
			types.MessageSystemAttributeNameApproximateReceiveCount,
			types.MessageSystemAttributeNameMessageGroupId,
		},
		MessageAttributeNames: []string{"All"},
	})
	if err != nil {
		return nil, normalizeError(err)
	}

	msgs := make([]worker.Message, 0, len(out.Messages))
	for _, m := range out.Messages {
		msgs = append(msgs, toMessage(m))
	}
	return msgs, nil
}

// Ack は、DeleteMessage でメッセージを削除します。
func (c *consumer) Ack(ctx context.Context, m worker.Message) error {
	ctx, endSpan := c.tracer.Start(ctx)
	defer endSpan()

	_, err := c.api.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(c.cfg.QueueURL),
		ReceiptHandle: aws.String(m.Attributes[worker.AttrReceiptHandle]),
	})
	return normalizeError(err)
}

// Nack は、メッセージを即時に再配送へ戻します。
func (c *consumer) Nack(ctx context.Context, m worker.Message) error {
	ctx, endSpan := c.tracer.Start(ctx)
	defer endSpan()

	_, err := c.api.ChangeMessageVisibility(ctx, &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(c.cfg.QueueURL),
		ReceiptHandle:     aws.String(m.Attributes[worker.AttrReceiptHandle]),
		VisibilityTimeout: 0,
	})
	return normalizeError(err)
}

// NackWithBackoff は、最低 d だけ遅延させてからメッセージを再配送へ戻します。
// d<=0 は Nack（即時）と等価です。
func (c *consumer) NackWithBackoff(ctx context.Context, m worker.Message, d time.Duration) error {
	if d <= 0 {
		return c.Nack(ctx, m)
	}

	ctx, endSpan := c.tracer.Start(ctx)
	defer endSpan()

	_, err := c.api.ChangeMessageVisibility(ctx, &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(c.cfg.QueueURL),
		ReceiptHandle:     aws.String(m.Attributes[worker.AttrReceiptHandle]),
		VisibilityTimeout: visibilitySeconds(d),
	})
	return normalizeError(err)
}

// Extend は、可視性タイムアウトを延長します（長時間 handler のハートビート）。
func (c *consumer) Extend(ctx context.Context, m worker.Message, d time.Duration) error {
	ctx, endSpan := c.tracer.Start(ctx)
	defer endSpan()

	_, err := c.api.ChangeMessageVisibility(ctx, &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(c.cfg.QueueURL),
		ReceiptHandle:     aws.String(m.Attributes[worker.AttrReceiptHandle]),
		VisibilityTimeout: visibilitySeconds(d),
	})
	return normalizeError(err)
}

// visibilitySeconds は、Duration を SQS の可視性秒数（int32）へ丸めます。
// サブ秒は切り捨てると 0 秒＝即時再配送になり処理中メッセージが重複配送されるため、
// 切り上げたうえで 1 秒を下限とします。
func visibilitySeconds(d time.Duration) int32 {
	return max(1, int32(math.Ceil(d.Seconds())))
}

// toMessage は、SQS のメッセージを broker 非依存の Message へ正規化します。
//   - MessageAttributes（traceparent 等）→ Attributes
//   - ReceiptHandle → Attributes の予約キー（engine は解釈しない）
//   - ApproximateReceiveCount → ReceiveCount / MessageGroupId → PartitionKey
func toMessage(m types.Message) worker.Message {
	attrs := make(map[string]string, len(m.MessageAttributes)+1)
	for k, v := range m.MessageAttributes {
		if v.StringValue != nil {
			attrs[k] = *v.StringValue
		}
	}
	if m.ReceiptHandle != nil {
		attrs[worker.AttrReceiptHandle] = *m.ReceiptHandle
	}

	receiveCount := 0
	if s, ok := m.Attributes[string(types.MessageSystemAttributeNameApproximateReceiveCount)]; ok {
		receiveCount, _ = strconv.Atoi(s)
	}

	return worker.Message{
		ID:           aws.ToString(m.MessageId),
		Body:         []byte(aws.ToString(m.Body)),
		Attributes:   attrs,
		ReceiveCount: receiveCount,
		PartitionKey: m.Attributes[string(types.MessageSystemAttributeNameMessageGroupId)],
	}
}
