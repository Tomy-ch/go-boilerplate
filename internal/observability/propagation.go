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

// extractFromCarrier は、prop を明示注入できる ExtractFromCarrier の中核です（テスト用 seam）。
func extractFromCarrier(ctx context.Context, attrs map[string]string, prop propagation.TextMapPropagator) context.Context {
	if len(attrs) == 0 {
		return ctx
	}
	return prop.Extract(ctx, mapCarrier(attrs))
}
