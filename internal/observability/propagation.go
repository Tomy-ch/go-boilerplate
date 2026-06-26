package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

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

// ExtractFromCarrier は、attrs（traceparent 等）からグローバル伝播器で trace context を継続します（D1）。
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

// InjectToCarrier は、現在の ctx の trace context（traceparent 等）をグローバル伝播器で attrs へ書き込みます（D1）。
// outbox emit 時に traceparent を headers へ載せ、後続の relay→受信側を起点 trace に繋ぐための公開ヘルパです。
func InjectToCarrier(ctx context.Context, attrs map[string]string) {
	injectToCarrier(ctx, attrs, otel.GetTextMapPropagator())
}

// injectToCarrier は、prop で ctx の trace context を attrs へ書き込みます。
func injectToCarrier(ctx context.Context, attrs map[string]string, prop propagation.TextMapPropagator) {
	if attrs == nil {
		return
	}
	prop.Inject(ctx, mapCarrier(attrs))
}
