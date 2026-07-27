package observability

import (
	"context"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"

	"go.opentelemetry.io/otel/trace"
)

// TraceContext は、トレースを識別するための情報を保持します。
type TraceContext struct {
	traceID      string
	spanID       string
	parentSpanID string
}

// ShouldLogWithSpan は、o11y モードが有効かつ ctx にアクティブな Span が存在するとき true を返す。span 付きログを出す前提条件チェックに使う。
func ShouldLogWithSpan(ctx context.Context, obsCfg *config.ObservabilityConfig) bool {
	return obsCfg.Enabled() && trace.SpanFromContext(ctx).SpanContext().IsValid()
}

// BuildSpanName は、レイヤー名、パッケージ名、関数名からスパン名を構築します。
func BuildSpanName(layer, pkgName, funcName string) string {
	return layer + delimiter + pkgName + delimiter + funcName
}

// ExtractTraceContext は、現在の span から traceID/spanID を抽出して返します（parentSpanID は設定しません）。
func ExtractTraceContext(ctx context.Context) *TraceContext {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return &TraceContext{}
	}
	spanCtx := span.SpanContext()
	return &TraceContext{
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
) (*TraceContext, context.Context, func()) {
	parentSC := trace.SpanFromContext(ctx).SpanContext()

	childCtx, span := tracer.tracer.Start(ctx, name, opts...)
	childSC := span.SpanContext()

	tc := &TraceContext{
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
func (tc *TraceContext) TraceID() string {
	return tc.traceID
}

// SpanID は、TraceContext から SpanID を取得します。
func (tc *TraceContext) SpanID() string {
	return tc.spanID
}

// ParentSpanID は、TraceContext から ParentSpanID を取得します。
func (tc *TraceContext) ParentSpanID() string {
	return tc.parentSpanID
}

// NewTraceExtractor は、ctx から trace_id / span_id を抽出する logging.TraceExtractor を返します。
// obs 無効時・ctx に有効な span が無い場合は 3 番目の戻り値 false を返し、Logger 側の trace 注入を
// 抑止します。
func NewTraceExtractor(obsCfg *config.ObservabilityConfig) logging.TraceExtractor {
	return func(ctx context.Context) (string, string, bool) {
		if !obsCfg.Enabled() {
			return "", "", false
		}
		spanCtx := trace.SpanFromContext(ctx).SpanContext()
		if !spanCtx.IsValid() {
			return "", "", false
		}
		return spanCtx.TraceID().String(), spanCtx.SpanID().String(), true
	}
}
