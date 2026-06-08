package observability

import (
	"context"

	"go-boilerplate/internal/config"

	"go.opentelemetry.io/otel/trace"
)

const (
	SpanEventStart = "start"
	SpanEventEnd   = "end"
)

// TraceContext は、トレースを識別するための情報を保持します。
type TraceContext struct {
	traceID      string
	spanID       string
	parentSpanID string
}

// ShouldLogWithSpan は、o11y モードと Span の有無から span 前提ログを出してよいか判定する公開ヘルパです。
// span 前提ログを足す際の拡張ポイントとして提供しており、現状は本番未結線（利用例は README 参照）。
func ShouldLogWithSpan(ctx context.Context, obsCfg *config.ObservabilityConfig) bool {
	return obsCfg.Enabled() && trace.SpanFromContext(ctx).SpanContext().IsValid()
}

// BuildSpanName は、レイヤー名、パッケージ名、関数名からスパン名を構築します。
func BuildSpanName(layer, pkgName, funcName string) string {
	return layer + delimiter + pkgName + delimiter + funcName
}

// ExtractTraceContext は、現在の span から traceID/spanID を抽出して返します（parentSpanID は設定しません）。
func ExtractTraceContext(ctx context.Context) TraceContext {
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

// StartSpanWithParent は、新しい子スパンを開始し、TraceContext、子コンテキスト、終了関数を返します。
func StartSpanWithParent(
	ctx context.Context,
	tracer LayerTracer,
	name string,
	opts ...trace.SpanStartOption,
) (TraceContext, context.Context, func()) {
	parentSC := trace.SpanFromContext(ctx).SpanContext()

	childCtx, span := tracer.tracer.Start(ctx, name, opts...)
	childSC := span.SpanContext()

	tc := TraceContext{
		traceID: childSC.TraceID().String(),
		spanID:  childSC.SpanID().String(),
	}

	if parentSC.IsValid() {
		tc.parentSpanID = parentSC.SpanID().String()
	}

	end := func() { span.End() }
	return tc, childCtx, end
}

// TraceID は、TraceContext から TraceID を取得します。
func (tc TraceContext) TraceID() string {
	return tc.traceID
}

// SpanID は、TraceContext から SpanID を取得します。
func (tc TraceContext) SpanID() string {
	return tc.spanID
}

// ParentSpanID は、TraceContext から ParentSpanID を取得します。
func (tc TraceContext) ParentSpanID() string {
	return tc.parentSpanID
}
