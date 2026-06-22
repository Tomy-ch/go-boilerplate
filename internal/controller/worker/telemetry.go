package worker

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/usecase/boundary/worker"
)

// attrCarrier は、Message.Attributes を propagation のキャリアとして扱うアダプタです。
type attrCarrier map[string]string

func (c attrCarrier) Get(key string) string { return c[key] }
func (c attrCarrier) Set(key, value string) { c[key] = value }
func (c attrCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// withTrace は、Message.Attributes の traceparent から trace context を継続します（D1）。
// producer → consumer → handler を 1 trace に繋ぎます。
func (r *run) withTrace(ctx context.Context, m worker.Message) context.Context {
	if len(m.Attributes) == 0 {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, attrCarrier(m.Attributes))
}

// extractTraceContext は、attrs の traceparent を prop で ctx に取り込みます（D1 の純粋部分・テスト用）。
func extractTraceContext(ctx context.Context, attrs map[string]string, prop propagation.TextMapPropagator) context.Context {
	if len(attrs) == 0 {
		return ctx
	}
	return prop.Extract(ctx, attrCarrier(attrs))
}

// msgFields は、構造化ログ用のフィールド（worker 名 / message id / receive count / trace id）を返します（D3）。
func msgFields(ctx context.Context, name string, m worker.Message) []*logging.Field {
	fields := []*logging.Field{
		logging.String(logging.WorkerNameKey, name),
		logging.String(logging.MessageIDKey, m.ID),
		logging.Int(logging.ReceiveCountKey, m.ReceiveCount),
	}
	if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
		fields = append(fields, logging.String(logging.TraceIDKey, sc.TraceID().String()))
	}
	return fields
}
