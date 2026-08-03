//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package user

import (
	"context"
	"time"

	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/domain/user"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/pkg/uuid"
)

const (
	// DefaultPurgeBatchSize は、1 バッチあたりの候補取得件数の既定値です。
	DefaultPurgeBatchSize int32 = 1_000
	// DefaultPurgeRetention は、退会したユーザーを物理削除せず保持する既定期間です。
	DefaultPurgeRetention = 30 * 24 * time.Hour
)

// PurgeResult は、物理削除の実行結果です。
type PurgeResult struct {
	// Purged は、物理削除したユーザー件数です。dryRun の場合は削除対象となった件数です。
	Purged int64
	// SkippedWithPurchases は、購入を保持しているため削除しなかったユーザー件数です。
	SkippedWithPurchases int64
}

// PurgeUsecase は、退会から retention を過ぎたユーザーを物理削除するユースケースを定義します。
type PurgeUsecase interface {
	// PurgeDeleted は、退会から retention より長く経過したユーザーを batchSize 件ずつ物理削除し、結果を返します。
	// 購入を持つユーザーは削除せず、その件数を結果に含めます。
	// dryRun が true の場合は削除を行わず、削除対象となった件数だけを結果に返します。
	// retention / batchSize が 0 以下の場合は、それぞれ既定値を用います。
	PurgeDeleted(ctx context.Context, retention time.Duration, batchSize int32, dryRun bool) (PurgeResult, error)
}

// purgeUsecase は、退会から retention を過ぎたユーザーを物理削除するユースケースを提供します。
type purgeUsecase struct {
	tracer       observability.LayerTracer
	txm          tx.Manager
	clock        clock.Clock
	userRepo     user.Repository
	purchaseRepo purchase.Repository
}

// purgeBatchResult は、1 バッチ分の物理削除の結果です。
type purgeBatchResult struct {
	// candidates は、このバッチで列挙した候補の件数です。batchSize に満たなければ最終バッチです。
	candidates int
	// purged は、このバッチで物理削除したユーザー件数です。dryRun の場合は削除対象となった件数です。
	purged int64
	// skipped は、購入を保持しているため削除しなかったユーザー件数です。
	skipped int64
	// nextAfter は、次バッチの列挙境界です。候補が無かった場合は nil です。
	nextAfter *uuid.UUID
}

// NewPurge は、PurgeUsecase を初期化します。
func NewPurge(
	tf observability.TracerFactory,
	txm tx.Manager,
	clock clock.Clock,
	userRepo user.Repository,
	purchaseRepo purchase.Repository,
) PurgeUsecase {
	return &purgeUsecase{
		tracer:       tf.Usecase(),
		txm:          txm,
		clock:        clock,
		userRepo:     userRepo,
		purchaseRepo: purchaseRepo,
	}
}

func (u *purgeUsecase) PurgeDeleted(
	ctx context.Context, retention time.Duration, batchSize int32, dryRun bool,
) (PurgeResult, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if retention <= 0 {
		retention = DefaultPurgeRetention
	}
	if batchSize <= 0 {
		batchSize = DefaultPurgeBatchSize
	}
	cutoff := u.clock.Now().Add(-retention)

	var (
		result  PurgeResult
		afterID *uuid.UUID
	)
	for {
		batch, err := u.purgeBatch(ctx, cutoff, afterID, batchSize, dryRun)
		if err != nil {
			return PurgeResult{}, err
		}
		result.Purged += batch.purged
		result.SkippedWithPurchases += batch.skipped

		// 候補がバッチを満たさなかった = これ以上対象は無い。
		if batch.candidates < int(batchSize) {
			return result, nil
		}
		// 削除しなかった候補を跨いで前進するため、境界は削除可否によらず候補の末尾まで進める。
		// ここを進めないと、先頭バッチが全件スキップ対象のとき同じ候補を取り直し続けてしまう。
		afterID = batch.nextAfter
	}
}

// purgeBatch は、1 バッチ分の候補列挙・購入照会・物理削除を 1 トランザクションで実行します。
func (u *purgeUsecase) purgeBatch(
	ctx context.Context, cutoff time.Time, afterID *uuid.UUID, batchSize int32, dryRun bool,
) (purgeBatchResult, error) {
	return tx.DoWithResult(ctx, u.txm, func(ctx context.Context) (purgeBatchResult, error) {
		ids, err := u.userRepo.FindDeletedBefore(ctx, cutoff, afterID, batchSize)
		if err != nil {
			return purgeBatchResult{}, err
		}
		if len(ids) == 0 {
			return purgeBatchResult{}, nil
		}

		withPurchases, err := u.purchaseRepo.FindUserIDsWithPurchases(ctx, ids)
		if err != nil {
			return purgeBatchResult{}, err
		}

		batch := purgeBatchResult{
			candidates: len(ids),
			skipped:    int64(len(withPurchases)),
			nextAfter:  &ids[len(ids)-1],
		}
		targets := excludeIDs(ids, withPurchases)
		if dryRun {
			batch.purged = int64(len(targets))
			return batch, nil
		}

		purged, err := u.userRepo.PurgeByIDs(ctx, targets)
		if err != nil {
			return purgeBatchResult{}, err
		}
		batch.purged = purged
		return batch, nil
	})
}

// excludeIDs は、ids から excluded に含まれる ID を取り除いた並びを返します。
func excludeIDs(ids, excluded []uuid.UUID) []uuid.UUID {
	if len(excluded) == 0 {
		return ids
	}
	excludedSet := make(map[uuid.UUID]struct{}, len(excluded))
	for _, id := range excluded {
		excludedSet[id] = struct{}{}
	}
	remained := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, ok := excludedSet[id]; !ok {
			remained = append(remained, id)
		}
	}
	return remained
}
