package worker

import (
	"context"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/worker"
)

// withTrace は、Message.Attributes の traceparent から trace context を継続します（D1）。
// producer → consumer → handler を 1 trace に繋ぎます。
func (r *run) withTrace(ctx context.Context, m worker.Message) context.Context {
	return observability.ExtractFromCarrier(ctx, m.Attributes)
}

// msgFields は、構造化ログ用のフィールド（worker 名 / message id / receive count）を返します（D3）。
// trace_id は Logger が ctx から自動注入するため、ここでは付与しません。
func msgFields(name string, m worker.Message) []*logging.Field {
	return []*logging.Field{
		logging.String(logging.WorkerNameKey, name),
		logging.String(logging.MessageIDKey, m.ID),
		logging.Int(logging.ReceiveCountKey, m.ReceiveCount),
	}
}
