package idempotency

import (
	"go-boilerplate/internal/usecase/boundary/clock"
	idempotencybndry "go-boilerplate/internal/usecase/boundary/idempotency"
	"go-boilerplate/internal/usecase/boundary/tx"
)

// NewDeps は、Run が必要とする依存を束ねて生成します。
// metrics が nil の場合はカウンタ操作が no-op になります。
func NewDeps(
	txm tx.Manager,
	store idempotencybndry.Store,
	clk clock.Clock,
	metrics Metrics,
) Deps {
	return Deps{
		Txm:     txm,
		Store:   store,
		Clock:   clk,
		Metrics: metrics,
	}
}
