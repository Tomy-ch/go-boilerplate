package sqs

import (
	"context"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/worker"
)

var _ worker.QueueStatsProvider = (*statsProvider)(nil)

// statsProvider は、worker.QueueStatsProvider の SQS 実装です。
// Consumer とは lifecycle を分離し、stats 収集のみを担います（API client / Config は共有）。
type statsProvider struct {
	api    API
	cfg    Config
	tracer observability.LayerTracer
}

// NewQueueStatsProvider は、SQS の滞留量を取得する QueueStatsProvider を生成します。
// NewConsumer とは別 capability として provide し、既存の Consumer interface 返しを変更しません。
func NewQueueStatsProvider(api API, cfg Config, tf observability.TracerFactory) worker.QueueStatsProvider {
	return &statsProvider{api: api, cfg: cfg, tracer: tf.Infra()}
}

// QueueStats は、source queue と（DLQURL があれば）DLQ の滞留量を取得します。
// DLQURL が空の場合は DLQ の取得をスキップし、DLQ は nil のままにします。
func (p *statsProvider) QueueStats(ctx context.Context) (worker.QueueStats, error) {
	ctx, endSpan := p.tracer.Start(ctx)
	defer endSpan()

	source, err := p.queueDepth(ctx, p.cfg.QueueURL)
	if err != nil {
		return worker.QueueStats{}, err
	}
	stats := worker.QueueStats{Source: source}

	if p.cfg.DLQURL != "" {
		dlq, err := p.queueDepth(ctx, p.cfg.DLQURL)
		if err != nil {
			return worker.QueueStats{}, err
		}
		stats.DLQ = &dlq
	}

	return stats, nil
}

// queueDepth は、GetQueueAttributes で approximate な滞留量を取得し QueueDepth へ正規化します。
func (p *statsProvider) queueDepth(ctx context.Context, queueURL string) (worker.QueueDepth, error) {
	out, err := p.api.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl: aws.String(queueURL),
		AttributeNames: []types.QueueAttributeName{
			types.QueueAttributeNameApproximateNumberOfMessages,
			types.QueueAttributeNameApproximateNumberOfMessagesNotVisible,
			types.QueueAttributeNameApproximateNumberOfMessagesDelayed,
		},
	})
	if err != nil {
		return worker.QueueDepth{}, normalizeError(err)
	}

	return worker.QueueDepth{
		Visible:  parseApproxCount(out.Attributes[string(types.QueueAttributeNameApproximateNumberOfMessages)]),
		InFlight: parseApproxCount(out.Attributes[string(types.QueueAttributeNameApproximateNumberOfMessagesNotVisible)]),
		Delayed:  parseApproxCount(out.Attributes[string(types.QueueAttributeNameApproximateNumberOfMessagesDelayed)]),
	}, nil
}

// parseApproxCount は、SQS の attribute 値を int64 へ変換します。
// attribute 欠落（空文字）や parse 不能は、滞留傾向の gauge として 0 件扱いにします。
func parseApproxCount(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
