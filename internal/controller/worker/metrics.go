package worker

import (
	"go.opentelemetry.io/otel/metric"

	"go-boilerplate/pkg/xerrors"
)

const meterName = "go-boilerplate/worker"

// metrics は、engine 所有（broker 非依存）の計装一式です（D2）。
// queue 滞留系（oldest-message-age / consumer-lag）は adapter 所有のため含みません。
type metrics struct {
	received   metric.Int64Counter
	processed  metric.Int64Counter
	failed     metric.Int64Counter
	retried    metric.Int64Counter
	dlq        metric.Int64Counter
	pollErrors metric.Int64Counter
	latencyMs  metric.Float64Histogram
	inFlight   metric.Int64UpDownCounter
}

// meterBuilder は、計装生成の最初のエラーを保持しつつ宣言的に組み立てます。
type meterBuilder struct {
	m   metric.Meter
	err error
}

// newMetrics は、worker engine の計装一式を生成します。いずれかの生成失敗で error を返します。
func newMetrics(m metric.Meter) (*metrics, error) {
	b := &meterBuilder{m: m}
	mt := &metrics{
		received:   b.counter("worker.received", "受信したメッセージ数"),
		processed:  b.counter("worker.processed", "正常処理したメッセージ数"),
		failed:     b.counter("worker.failed", "処理に失敗したメッセージ数"),
		retried:    b.counter("worker.retried", "Nack して再配送へ戻したメッセージ数"),
		dlq:        b.counter("worker.dlq", "FailureHandler へ退避したメッセージ数"),
		pollErrors: b.counter("worker.poll_errors", "Receive のエラー回数"),
		latencyMs:  b.histogram("worker.processing_latency_ms", "Handle の処理時間(ミリ秒)"),
		inFlight:   b.upDownCounter("worker.in_flight", "処理中(受信済み・未確定)のメッセージ数"),
	}
	if b.err != nil {
		return nil, b.err
	}
	return mt, nil
}

func (b *meterBuilder) counter(name, desc string) metric.Int64Counter {
	if b.err != nil {
		return nil
	}
	c, err := b.m.Int64Counter(name, metric.WithDescription(desc))
	if err != nil {
		b.err = xerrors.Wrap(err, name)
		return nil
	}
	return c
}

func (b *meterBuilder) histogram(name, desc string) metric.Float64Histogram {
	if b.err != nil {
		return nil
	}
	h, err := b.m.Float64Histogram(name, metric.WithDescription(desc), metric.WithUnit("ms"))
	if err != nil {
		b.err = xerrors.Wrap(err, name)
		return nil
	}
	return h
}

func (b *meterBuilder) upDownCounter(name, desc string) metric.Int64UpDownCounter {
	if b.err != nil {
		return nil
	}
	c, err := b.m.Int64UpDownCounter(name, metric.WithDescription(desc))
	if err != nil {
		b.err = xerrors.Wrap(err, name)
		return nil
	}
	return c
}
