package observability

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// outboxMeterName は、outbox relay 計装の meter 名です。
const outboxMeterName = "go-boilerplate/outbox"

// OutboxMetrics は、outbox relay 所有の outbox 固有計装です。
// publish 自体の count/latency/error は httpclient(downstream=outbox) の計装で賄うため、
// ここでは outbox 固有の SLI（lag / blocked streams）と dead 数のみを持ちます。
// relay は配送チャネルごとに独立して動くため、いずれもチャネル属性を伴います。
type OutboxMetrics struct {
	lagSeconds     metric.Int64Gauge
	dead           metric.Int64Counter
	blockedStreams metric.Int64Gauge
}

// NewOutboxMetrics は、注入された MeterProvider から outbox relay の計装一式を生成します。
func NewOutboxMetrics(mp metric.MeterProvider) (*OutboxMetrics, error) {
	b := &meterBuilder{m: mp.Meter(outboxMeterName)}
	om := &OutboxMetrics{
		lagSeconds:     b.gauge("outbox.lag_seconds", "最古 pending 行の経過秒数（SLI=outbox lag）"),
		dead:           b.counter("outbox.dead", "恒久エラーと判定して dead 化したメッセージ数"),
		blockedStreams: b.gauge("outbox.blocked_streams", "先頭行が dead で進行が止まっているストリーム数"),
	}
	if b.err != nil {
		return nil, b.err
	}
	return om, nil
}

// SetLagSeconds は、最古 pending 行の経過秒数（outbox lag）をチャネル別に記録します（pending 無しは 0）。
func (m *OutboxMetrics) SetLagSeconds(ctx context.Context, channel string, seconds int64) {
	m.lagSeconds.Record(ctx, seconds, metric.WithAttributes(attribute.String("channel", channel)))
}

// IncDead は、dead 化したメッセージ数をチャネル別に計上します。
func (m *OutboxMetrics) IncDead(ctx context.Context, channel string) {
	m.dead.Add(ctx, 1, metric.WithAttributes(attribute.String("channel", channel)))
}

// SetBlockedStreams は、先頭行が dead で止まっているストリーム数をチャネル別に記録します。
func (m *OutboxMetrics) SetBlockedStreams(ctx context.Context, channel string, count int64) {
	m.blockedStreams.Record(ctx, count, metric.WithAttributes(attribute.String("channel", channel)))
}
