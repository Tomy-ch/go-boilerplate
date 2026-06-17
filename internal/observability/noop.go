package observability

import (
	"context"

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
