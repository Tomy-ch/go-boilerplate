package observability

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

// NewHTTPClientTransport は、外部 HTTP client substrate 用の otelhttp 計装済み RoundTripper を生成します。
//
// otelpgx を pgxpool に結線するのと対称に、HTTP span 生成と W3C traceparent の outgoing inject を
// 自動化します。RED metrics は HTTPClientMetrics が Downstream 単位で担うため、otelhttp の自動 metrics
// は no-op MeterProvider を渡して無効化し、二重計上を防ぎます（trace のみ利用）。
func NewHTTPClientTransport(tp trace.TracerProvider) http.RoundTripper {
	base := http.DefaultTransport
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		base = t.Clone()
	}
	return otelhttp.NewTransport(
		base,
		otelhttp.WithTracerProvider(tp),
		otelhttp.WithMeterProvider(metricnoop.NewMeterProvider()),
	)
}
