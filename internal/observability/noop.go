package observability

import (
	"context"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// noopSpanExporter は span を送出しない SpanExporter。
type noopSpanExporter struct{}

// newNoopSpanExporter は送出先未指定時のフォールバック SpanExporter を返す。
func newNoopSpanExporter(context.Context) (sdktrace.SpanExporter, error) {
	return noopSpanExporter{}, nil
}

func (noopSpanExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error { return nil }

func (noopSpanExporter) Shutdown(context.Context) error { return nil }

// newNoopMetricReader は送出先未指定時のフォールバック MetricReader を返す(ManualReader は no-op)。
func newNoopMetricReader(context.Context) (sdkmetric.Reader, error) {
	return sdkmetric.NewManualReader(), nil
}

// isNoopSpanExporter は、送出を行わない SpanExporter かを判定する。
// env 未指定時のフォールバック(自前の noopSpanExporter)と、OTEL_TRACES_EXPORTER=none 指定時に
// autoexport が返す no-op の両方を no-op とみなす。
func isNoopSpanExporter(e sdktrace.SpanExporter) bool {
	if _, ok := e.(noopSpanExporter); ok {
		return true
	}
	return autoexport.IsNoneSpanExporter(e)
}

// isNoopMetricReader は、計測を行わない MetricReader かを判定する。
// env 未指定時のフォールバック(自前の ManualReader)と、OTEL_METRICS_EXPORTER=none 指定時に
// autoexport が返す no-op の両方を no-op とみなす。
func isNoopMetricReader(r sdkmetric.Reader) bool {
	if _, ok := r.(*sdkmetric.ManualReader); ok {
		return true
	}
	return autoexport.IsNoneMetricReader(r)
}
