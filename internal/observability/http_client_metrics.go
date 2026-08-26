package observability

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// httpClientMeterName は、外部 HTTP client substrate 計装の meter 名です。
const httpClientMeterName = "go-boilerplate/httpclient"

// HTTPClientMetrics は、外部 HTTP client の計装一式です
// （RED＝リクエスト数・エラー数・レイテンシに、retry 回数・処理中リクエスト数・breaker 状態 gauge を加えたもの）。
//
// downstream には registry で解決される固定の Downstream 名など低カーディナリティな識別子のみを
// 渡します。生の URL やリクエスト固有の値を渡すとメトリクスのカーディナリティが際限なく増えます。
type HTTPClientMetrics struct {
	requests     metric.Int64Counter
	errors       metric.Int64Counter
	retries      metric.Int64Counter
	latencyMs    metric.Float64Histogram
	inFlight     metric.Int64UpDownCounter
	breakerState metric.Int64Gauge
}

// NewHTTPClientMetrics は、注入された MeterProvider から substrate の計装一式を生成します。
// いずれかの計装生成失敗で error を返します。
func NewHTTPClientMetrics(mp metric.MeterProvider) (*HTTPClientMetrics, error) {
	b := &meterBuilder{m: mp.Meter(httpClientMeterName)}
	hm := &HTTPClientMetrics{
		requests:     b.counter("httpclient.requests", "完了した外部 HTTP リクエスト数(ステータスクラス別)"),
		errors:       b.counter("httpclient.errors", "失敗した外部 HTTP リクエスト数(理由別)"),
		retries:      b.counter("httpclient.retries", "リトライした回数"),
		latencyMs:    b.histogram("httpclient.request_latency_ms", "リクエスト全体の所要時間(ミリ秒)"),
		inFlight:     b.upDownCounter("httpclient.in_flight", "処理中(送信済み・未完了)のリクエスト数"),
		breakerState: b.gauge("httpclient.breaker_state", "circuit breaker の状態(0:closed 1:half-open 2:open)"),
	}
	if b.err != nil {
		return nil, b.err
	}
	return hm, nil
}

// RecordRequest は、完了したリクエストを Downstream とステータスクラス別に計上します。
func (m *HTTPClientMetrics) RecordRequest(ctx context.Context, downstream, statusClass string) {
	m.requests.Add(ctx, 1, metric.WithAttributes(
		attribute.String("downstream", downstream),
		attribute.String("status_class", statusClass),
	))
}

// RecordError は、失敗したリクエストを Downstream と理由別に計上します。
func (m *HTTPClientMetrics) RecordError(ctx context.Context, downstream, reason string) {
	m.errors.Add(ctx, 1, metric.WithAttributes(
		attribute.String("downstream", downstream),
		attribute.String("reason", reason),
	))
}

// RecordRetry は、リトライ回数を Downstream 別に計上します。
func (m *HTTPClientMetrics) RecordRetry(ctx context.Context, downstream string) {
	m.retries.Add(ctx, 1, metric.WithAttributes(attribute.String("downstream", downstream)))
}

// RecordLatencyMs は、リクエスト全体の所要時間(ミリ秒)を Downstream 別に記録します。
func (m *HTTPClientMetrics) RecordLatencyMs(ctx context.Context, downstream string, ms float64) {
	m.latencyMs.Record(ctx, ms, metric.WithAttributes(attribute.String("downstream", downstream)))
}

// InFlightAdd は、処理中のリクエスト数を Downstream 別に増減します。
func (m *HTTPClientMetrics) InFlightAdd(ctx context.Context, downstream string, delta int64) {
	m.inFlight.Add(ctx, delta, metric.WithAttributes(attribute.String("downstream", downstream)))
}

// SetBreakerState は、circuit breaker の状態を Downstream 別に記録します（0:closed 1:half-open 2:open）。
func (m *HTTPClientMetrics) SetBreakerState(ctx context.Context, downstream string, state int64) {
	m.breakerState.Record(ctx, state, metric.WithAttributes(attribute.String("downstream", downstream)))
}
