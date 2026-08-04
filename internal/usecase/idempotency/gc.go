//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE
package idempotency

import (
	"context"

	"go-boilerplate/internal/usecase/boundary/clock"
	idempotencybndry "go-boilerplate/internal/usecase/boundary/idempotency"
)

// DefaultGCBatchSize は、GC の 1 バッチあたり削除件数の既定値です。
const DefaultGCBatchSize int32 = 10_000

// GCMetrics は、GC の o11y カウンタです。GCUsecase は GC 固有カウンタのみ必要なため
// interface を分離しています。
// 各メソッドは ctx を第 1 引数に取り、OTel exemplar（メトリクス→トレース相関）を維持します。
type GCMetrics interface {
	// IncExpiredCleanup は、削除に成功した失効キー件数を計上します。
	IncExpiredCleanup(ctx context.Context, count int64)
	// IncExpiredCleanupFailure は、削除バッチの失敗回数を計上します。
	IncExpiredCleanupFailure(ctx context.Context)
}

// GCUsecase は、TTL 失効した冪等性キーを掃除するユースケースです。
type GCUsecase interface {
	// SweepExpired は、失効したエントリを batchSize 件ずつ削除し、合計削除件数を返します。
	// エラーを返す場合も、失敗したバッチより前にコミット済みの件数を返します。
	// コミット済みの削除は取り消せないため、呼び手が実績を失わないようにするためです。
	SweepExpired(ctx context.Context, batchSize int32) (int64, error)
}

type gcUsecase struct {
	store idempotencybndry.Store
	clock clock.Clock
	// metricsImpl は任意。nil の場合はすべてのカウンタ操作が no-op になります。
	metricsImpl GCMetrics
}

type nopGCMetrics struct{}

// NewGC は、GCUsecase を生成します。metrics が nil の場合はカウンタ操作が no-op になります。
func NewGC(store idempotencybndry.Store, clk clock.Clock, metrics GCMetrics) GCUsecase {
	return &gcUsecase{store: store, clock: clk, metricsImpl: metrics}
}

func (g *gcUsecase) SweepExpired(ctx context.Context, batchSize int32) (int64, error) {
	if batchSize <= 0 {
		batchSize = DefaultGCBatchSize
	}
	cutoff := g.clock.Now()

	var total int64
	for {
		deleted, err := g.store.DeleteExpired(ctx, cutoff, batchSize)
		if err != nil {
			g.metrics().IncExpiredCleanupFailure(ctx)
			return total, err
		}
		if deleted > 0 {
			g.metrics().IncExpiredCleanup(ctx, deleted)
		}
		total += deleted
		// バッチが満たなかった = もう失効したエントリは無い。
		if deleted < int64(batchSize) {
			return total, nil
		}
	}
}

func (g *gcUsecase) metrics() GCMetrics {
	if g.metricsImpl == nil {
		return nopGCMetrics{}
	}
	return g.metricsImpl
}

func (nopGCMetrics) IncExpiredCleanup(context.Context, int64) {}
func (nopGCMetrics) IncExpiredCleanupFailure(context.Context) {}
