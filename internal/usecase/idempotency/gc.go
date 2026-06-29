//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE
package idempotency

import (
	"context"

	"go-boilerplate/internal/usecase/boundary/clock"
	idempotencybndry "go-boilerplate/internal/usecase/boundary/idempotency"
)

// DefaultGCBatchSize は、GC の 1 バッチあたり削除件数の既定値です。
const DefaultGCBatchSize int32 = 10_000

// GCUsecase は、TTL 失効した冪等性キーを掃除するユースケースです。
type GCUsecase interface {
	// SweepExpired は、失効行を batchSize 件ずつ削除し、合計削除件数を返します。
	SweepExpired(ctx context.Context, batchSize int32) (int64, error)
}

type gcUsecase struct {
	store idempotencybndry.Store
	clock clock.Clock
}

// NewGC は、GCUsecase を生成します。
func NewGC(store idempotencybndry.Store, clk clock.Clock) GCUsecase {
	return &gcUsecase{store: store, clock: clk}
}

// SweepExpired は、失効行を batchSize 件ずつ削除し、合計削除件数を返します。
func (g *gcUsecase) SweepExpired(ctx context.Context, batchSize int32) (int64, error) {
	if batchSize <= 0 {
		batchSize = DefaultGCBatchSize
	}
	cutoff := g.clock.Now()

	var total int64
	for {
		deleted, err := g.store.DeleteExpired(ctx, cutoff, batchSize)
		if err != nil {
			return total, err
		}
		total += deleted
		// バッチが満たなかった = もう失効行は無い。
		if deleted < int64(batchSize) {
			return total, nil
		}
	}
}
