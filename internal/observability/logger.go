package observability

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// logFields は、ログ出力用のフィールドを生成します。
func (lt LayerTracer) logFields(
	ctx context.Context,
	layer, pkgName, funcName string,
	spanEvent, spanName string,
	extra ...zap.Field,
) []zap.Field {
	fields := []zap.Field{
		zap.String("span_event", spanEvent),
		zap.String("span_name", spanName),
		zap.String("layer", layer),
		zap.String("package", pkgName),
		zap.String("func", funcName),
	}

	spanCtx := trace.SpanFromContext(ctx).SpanContext()
	if spanCtx.HasTraceID() {
		fields = append(fields,
			zap.String("trace_id", spanCtx.TraceID().String()),
			zap.String("span_id", spanCtx.SpanID().String()),
		)
	}

	if len(extra) > 0 {
		fields = append(fields, extra...)
	}

	return fields
}
