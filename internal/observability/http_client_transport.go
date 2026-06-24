package observability

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"syscall"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// HTTPClientTransport は、otelhttp 計装済みの outbound transport を包む不透明な型です。
//
// substrate（httpclient）の公開 API に net/http 型を露出させないためのラッパで、
// otel-contrib への依存も本パッケージに閉じ込めます（otelpgx を driver へ渡すのと対称）。
type HTTPClientTransport struct {
	rt http.RoundTripper
}

// tracePropagationKey は、outbound 呼び出しでトレース伝搬を行うかのフラグを保持する ctx キーです。
type tracePropagationKey struct{}

// conditionalPropagator は、ctx のフラグが false のとき Inject を抑止する propagator です。
type conditionalPropagator struct {
	inner propagation.TextMapPropagator
}

// NewHTTPClientTransport は、SSRF ガード付き base transport を otelhttp で計装した outbound transport を生成します。
//
// HTTP span 生成は自動化しますが、traceparent/baggage の outgoing inject は ContextWithTracePropagation の
// フラグに従い、信頼できない外部 downstream への伝搬を抑止できます。RED metrics は HTTPClientMetrics が
// 担うため otelhttp 自動 metrics は no-op MeterProvider で無効化します。
func NewHTTPClientTransport(tp trace.TracerProvider) *HTTPClientTransport {
	rt := otelhttp.NewTransport(
		newGuardedBaseTransport(),
		otelhttp.WithTracerProvider(tp),
		otelhttp.WithMeterProvider(metricnoop.NewMeterProvider()),
		otelhttp.WithPropagators(conditionalPropagator{inner: otel.GetTextMapPropagator()}),
	)
	return &HTTPClientTransport{rt: rt}
}

// RoundTripper は、内部の RoundTripper を返します（substrate 内部でのみ利用します）。
func (t *HTTPClientTransport) RoundTripper() http.RoundTripper {
	return t.rt
}

// ContextWithTracePropagation は、この outbound 呼び出しで traceparent/baggage を注入するかを ctx に設定します。
func ContextWithTracePropagation(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, tracePropagationKey{}, enabled)
}

// Inject は、伝搬フラグが明示的に false の場合のみ注入を抑止し、それ以外は内側へ委譲します。
func (p conditionalPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	if enabled, ok := ctx.Value(tracePropagationKey{}).(bool); ok && !enabled {
		return
	}
	p.inner.Inject(ctx, carrier)
}

// Extract は、内側の propagator へ委譲します。
func (p conditionalPropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return p.inner.Extract(ctx, carrier)
}

// Fields は、内側の propagator へ委譲します。
func (p conditionalPropagator) Fields() []string {
	return p.inner.Fields()
}

// newGuardedBaseTransport は、リンクローカル宛て接続を拒否する DialContext を備えた base transport を返します。
func newGuardedBaseTransport() *http.Transport {
	base := &http.Transport{}
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		base = t.Clone()
	}
	dialer := &net.Dialer{Control: ssrfGuardControl}
	base.DialContext = dialer.DialContext
	return base
}

// ssrfGuardControl は、リンクローカル（クラウドメタデータ 169.254.169.254 等）への接続を拒否します。
// 名前解決後の実接続先 IP を検査するため、DNS rebinding を含む SSRF を防ぎます。
func ssrfGuardControl(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip != nil && (ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()) {
		return fmt.Errorf("ssrf guard: blocked link-local address %s", host)
	}
	return nil
}
