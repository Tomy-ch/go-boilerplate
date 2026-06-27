package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// traceContextPropagator は、traceparent/tracestate のみを伝搬する propagator です（Baggage を含まない）。
var traceContextPropagator propagation.TextMapPropagator = propagation.TraceContext{}

// mapCarrier は、map[string]string を propagation のキャリアとして扱うアダプタです。
type mapCarrier map[string]string

func (c mapCarrier) Get(key string) string { return c[key] }
func (c mapCarrier) Set(key, value string) { c[key] = value }
func (c mapCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// ExtractFromCarrier は、attrs（traceparent 等）からグローバル伝播器で trace context を継続します。
// producer → consumer → handler を 1 trace に繋ぐための公開ヘルパです。
func ExtractFromCarrier(ctx context.Context, attrs map[string]string) context.Context {
	return extractFromCarrier(ctx, attrs, otel.GetTextMapPropagator())
}

// extractFromCarrier は、prop から trace context を抽出して ctx に付与します。
func extractFromCarrier(ctx context.Context, attrs map[string]string, prop propagation.TextMapPropagator) context.Context {
	if len(attrs) == 0 {
		return ctx
	}
	return prop.Extract(ctx, mapCarrier(attrs))
}

// InjectTraceContextToCarrier は、現在の ctx の trace context（traceparent 等）のみを attrs へ書き込みます。
// outbox emit 時に traceparent を headers へ載せ、後続の relay→受信側を起点 trace に繋ぐための公開ヘルパです。
// グローバル伝播器（Baggage を含みうる）ではなく TraceContext 限定で inject することで、インバウンド由来の
// 任意 baggage が outbox 経由で外部エンドポイントへ転送される経路を断ちます。
func InjectTraceContextToCarrier(ctx context.Context, attrs map[string]string) {
	injectToCarrier(ctx, attrs, traceContextPropagator)
}

// injectToCarrier は、prop で ctx の trace context を attrs へ書き込みます。
func injectToCarrier(ctx context.Context, attrs map[string]string, prop propagation.TextMapPropagator) {
	if attrs == nil {
		return
	}
	prop.Inject(ctx, mapCarrier(attrs))
}
