package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// maxTraceStateBytes は、持ち回る tracestate の上限です（W3C の推奨上限）。
// tracestate は client が任意に注入でき、32 member × 512 バイトまで文法上は通ります。
// link を張るのに必要なのは traceparent だけなので、超えた分は運ばずに捨てます。
const maxTraceStateBytes = 512

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

// TraceContextFromCarrier は、attrs から trace context の項目だけを取り出した carrier を返します。
// 非同期のホップで起点 trace を持ち回るとき、運ぶ側が項目名を知らずに済むようにするためのものです。
// 該当が無ければ nil を返します。
func TraceContextFromCarrier(attrs map[string]string) map[string]string {
	if len(attrs) == 0 {
		return nil
	}

	out := map[string]string{}
	for _, key := range traceContextPropagator.Fields() {
		value, ok := attrs[key]
		if !ok || value == "" || len(value) > maxTraceStateBytes {
			continue
		}

		out[key] = value
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

// InjectTraceContextToCarrier は、現在の ctx の trace context（traceparent / tracestate）だけを attrs へ書き込みます。
// グローバル伝播器ではなく TraceContext 限定なのは意図したもので、インバウンド由来の baggage を
// 外部へ運ばないためです（README「Trace Context Propagation」）。
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
