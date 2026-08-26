package observability

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

// workerMeterName は、worker engine 計装の meter 名です。
const workerMeterName = "go-boilerplate/worker"

// WorkerMetrics は、worker engine 所有（broker 非依存）の計装一式です。
type WorkerMetrics struct {
	received     metric.Int64Counter
	processed    metric.Int64Counter
	failed       metric.Int64Counter
	retried      metric.Int64Counter
	dlq          metric.Int64Counter
	pollErrors   metric.Int64Counter
	extendErrors metric.Int64Counter
	latencyMs    metric.Float64Histogram
	inFlight     metric.Int64UpDownCounter
}

// NewWorkerMetrics は、注入された MeterProvider から worker engine の計装一式を生成します。
// いずれかの計装生成失敗で error を返します。
func NewWorkerMetrics(mp metric.MeterProvider) (*WorkerMetrics, error) {
	b := &meterBuilder{m: mp.Meter(workerMeterName)}
	wm := &WorkerMetrics{
		received:     b.counter("worker.received", "受信したメッセージ数"),
		processed:    b.counter("worker.processed", "正常処理したメッセージ数"),
		failed:       b.counter("worker.failed", "処理に失敗したメッセージ数"),
		retried:      b.counter("worker.retried", "Nack して再配送へ戻したメッセージ数"),
		dlq:          b.counter("worker.dlq", "FailureHandler へ退避したメッセージ数"),
		pollErrors:   b.counter("worker.poll_errors", "Receive のエラー回数"),
		extendErrors: b.counter("worker.extend_errors", "Extend(ハートビート)のエラー回数"),
		latencyMs:    b.histogram("worker.processing_latency_ms", "Handle の処理時間(ミリ秒)"),
		inFlight:     b.upDownCounter("worker.in_flight", "処理中(受信済み・未確定)のメッセージ数"),
	}
	if b.err != nil {
		return nil, b.err
	}
	return wm, nil
}

// Received は、受信したメッセージ数を計上します。
func (m *WorkerMetrics) Received(ctx context.Context, n int64) { m.received.Add(ctx, n) }

// Processed は、正常処理したメッセージ数を計上します。
func (m *WorkerMetrics) Processed(ctx context.Context) { m.processed.Add(ctx, 1) }

// Failed は、処理に失敗したメッセージ数を計上します。
func (m *WorkerMetrics) Failed(ctx context.Context) { m.failed.Add(ctx, 1) }

// Retried は、Nack して再配送へ戻したメッセージ数を計上します。
func (m *WorkerMetrics) Retried(ctx context.Context) { m.retried.Add(ctx, 1) }

// DLQ は、FailureHandler へ退避したメッセージ数を計上します。
func (m *WorkerMetrics) DLQ(ctx context.Context) { m.dlq.Add(ctx, 1) }

// PollError は、Receive のエラー回数を計上します。
func (m *WorkerMetrics) PollError(ctx context.Context) { m.pollErrors.Add(ctx, 1) }

// ExtendError は、Extend(ハートビート)のエラー回数を計上します。
func (m *WorkerMetrics) ExtendError(ctx context.Context) { m.extendErrors.Add(ctx, 1) }

// RecordLatencyMs は、Handle の処理時間(ミリ秒)を記録します。
func (m *WorkerMetrics) RecordLatencyMs(ctx context.Context, ms float64) { m.latencyMs.Record(ctx, ms) }

// InFlightAdd は、処理中(受信済み・未確定)のメッセージ数を増減します。
func (m *WorkerMetrics) InFlightAdd(ctx context.Context, delta int64) { m.inFlight.Add(ctx, delta) }
