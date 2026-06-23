package idempotency

import (
	"go-boilerplate/internal/usecase/boundary/clock"
	idempotencybndry "go-boilerplate/internal/usecase/boundary/idempotency"
	"go-boilerplate/internal/usecase/boundary/tx"
)

// NewDeps は、Run が必要とする依存を束ねて生成します。
func NewDeps(
	txm tx.Manager,
	store idempotencybndry.Store,
	clk clock.Clock,
) Deps {
	return Deps{
		Txm:   txm,
		Store: store,
		Clock: clk,
	}
}
