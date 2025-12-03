package observability

import (
	"context"

	"boilerplate-go/internal/config"

	"go.opentelemetry.io/otel/trace"
)

const (
	SpanEventStart = "start"
	SpanEventEnd   = "end"
)

// TraceContext は、トレースを識別するための情報を保持します。
type TraceContext struct {
	traceID string
	spanID  string
}

// ShouldLogWithSpan は、o11yモードとSpanの有無から、「このログを span 前提で出してよいか」を判定します。
func ShouldLogWithSpan(ctx context.Context, obsCfg *config.ObservabilityConfig) bool {
	return obsCfg.Enabled() && trace.SpanFromContext(ctx).SpanContext().IsValid()
}

// BuildSpanName は、レイヤー名、パッケージ名、関数名からスパン名を構築します。
func BuildSpanName(layer, pkgName, funcName string) string {
	return layer + delimiter + pkgName + delimiter + funcName
}

// ExtractSpan は、Context からトレース情報を抽出して返します。
func ExtractSpan(ctx context.Context) TraceContext {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return TraceContext{}
	}
	spanCtx := span.SpanContext()
	return TraceContext{
		traceID: spanCtx.TraceID().String(),
		spanID:  spanCtx.SpanID().String(),
	}
}

// TraceID は、TraceContext から TraceID を取得します。
func (tc TraceContext) TraceID() string {
	return tc.traceID
}

// SpanID は、TraceContext から SpanID を取得します。
func (tc TraceContext) SpanID() string {
	return tc.spanID
}
