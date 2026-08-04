//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package outbox

import (
	"context"
	"time"

	"go-boilerplate/internal/usecase/boundary/clock"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
)

const (
	// DefaultGCBatchSize は、GC の 1 バッチあたり削除件数の既定値です。
	DefaultGCBatchSize int32 = 10_000
	// DefaultRetention は、published のエントリを保持する既定期間です。これより古いエントリが GC 対象になります。
	DefaultRetention = 7 * 24 * time.Hour
)

// GCUsecase は、published 済みの outbox エントリを刈り取るユースケースです。
type GCUsecase interface {
	// SweepPublished は、retention より古い published エントリを batchSize 件ずつ削除し、合計削除件数を返します。
	// エラーを返す場合も、失敗したバッチより前にコミット済みの件数を返します。
	// コミット済みの削除は取り消せないため、呼び手が実績を失わないようにするためです。
	SweepPublished(ctx context.Context, batchSize int32) (int64, error)
}

type gcUsecase struct {
	store     outboxbndry.Store
	clock     clock.Clock
	retention time.Duration
}

// NewGC は、GCUsecase を生成します。
func NewGC(store outboxbndry.Store, clk clock.Clock) GCUsecase {
	return &gcUsecase{store: store, clock: clk, retention: DefaultRetention}
}

func (g *gcUsecase) SweepPublished(ctx context.Context, batchSize int32) (int64, error) {
	if batchSize <= 0 {
		batchSize = DefaultGCBatchSize
	}
	cutoff := g.clock.Now().Add(-g.retention)

	var total int64
	for {
		deleted, err := g.store.DeletePublished(ctx, cutoff, batchSize)
		if err != nil {
			return total, err
		}
		total += deleted
		// バッチが満たなかった = もう対象のエントリは無い。
		if deleted < int64(batchSize) {
			return total, nil
		}
	}
}
