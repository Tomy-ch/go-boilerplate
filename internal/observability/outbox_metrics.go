package observability

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

// outboxMeterName は、outbox relay 計装の meter 名です。
const outboxMeterName = "go-boilerplate/outbox"

// OutboxMetrics は、outbox relay 所有の outbox 固有計装です。
// publish 自体の count/latency/error は httpclient(downstream=outbox) の計装で賄うため、
// ここでは outbox 固有の SLI（lag）と dead 数のみを持ちます。
type OutboxMetrics struct {
	lagSeconds metric.Int64Gauge
	dead       metric.Int64Counter
}

// NewOutboxMetrics は、注入された MeterProvider から outbox relay の計装一式を生成します。
func NewOutboxMetrics(mp metric.MeterProvider) (*OutboxMetrics, error) {
	b := &meterBuilder{m: mp.Meter(outboxMeterName)}
	om := &OutboxMetrics{
		lagSeconds: b.gauge("outbox.lag_seconds", "最古 pending 行の経過秒数（SLI=outbox lag）"),
		dead:       b.counter("outbox.dead", "max attempts 到達で dead 化したメッセージ数"),
	}
	if b.err != nil {
		return nil, b.err
	}
	return om, nil
}

// SetLagSeconds は、最古 pending 行の経過秒数（outbox lag）を記録します（pending 無しは 0）。
func (m *OutboxMetrics) SetLagSeconds(ctx context.Context, seconds int64) {
	m.lagSeconds.Record(ctx, seconds)
}

// IncDead は、dead 化したメッセージ数を計上します。
func (m *OutboxMetrics) IncDead(ctx context.Context) { m.dead.Add(ctx, 1) }
