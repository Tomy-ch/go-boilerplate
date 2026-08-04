//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package product

import (
	"context"
	"strings"
	"time"

	"go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/internal/usecase/boundary/objectstorage"
)

const (
	// DefaultImageGCBatchSize は、1 バッチあたりのオブジェクト列挙件数の既定値です。
	DefaultImageGCBatchSize int32 = 1_000
	// DefaultImageGCGrace は、アップロードされた画像を未参照でも回収しない既定の猶予期間です。
	// アップロード直後のオブジェクトは「商品作成フォーム記入中でまだ参照されていない」正常な状態と
	// 区別がつかないため、管理画面のフォーム滞在時間を見込んで待ちます。
	DefaultImageGCGrace = 24 * time.Hour
)

// ImageGCResult は、未参照画像の回収結果です。
type ImageGCResult struct {
	// Deleted は、削除したオブジェクト件数です。dryRun の場合は削除対象となった件数です。
	Deleted int64
	// Scanned は、猶予期間を過ぎており参照の有無を照合したオブジェクト件数です。
	Scanned int64
}

// ImageGCUsecase は、どの商品からも参照されていない画像オブジェクトを回収するユースケースを定義します。
type ImageGCUsecase interface {
	// SweepOrphans は、アップロードから grace より長く経過し、どの商品からも参照されていない
	// 商品画像オブジェクトを batchSize 件ずつ削除し、結果を返します。
	// dryRun が true の場合は削除を行わず、削除対象となった件数だけを結果に返します。
	// grace / batchSize が 0 以下の場合は、それぞれ既定値を用います。
	// エラーを返す場合も、失敗したページより前に削除済みの累計を結果に含めます。
	// 削除済みのオブジェクトは復元できないため、呼び手が実績を失わないようにするためです。
	SweepOrphans(ctx context.Context, grace time.Duration, batchSize int32, dryRun bool) (ImageGCResult, error)
}

// imageGCUsecase は、未参照の商品画像オブジェクトを回収するユースケースを提供します。
type imageGCUsecase struct {
	tracer      observability.LayerTracer
	clock       clock.Clock
	storage     objectstorage.Storage
	productRepo product.Repository
}

// imageGCPageResult は、1 ページ分の回収結果です。
type imageGCPageResult struct {
	// scanned は、このページで参照の有無を照合したオブジェクト件数です。
	scanned int64
	// deleted は、このページで削除したオブジェクト件数です。dryRun の場合は削除対象となった件数です。
	deleted int64
	// nextCursor は、次ページの列挙境界です。空なら最終ページです。
	nextCursor string
}

// NewImageGC は、ImageGCUsecase を初期化します。
func NewImageGC(
	tf observability.TracerFactory,
	clk clock.Clock,
	storage objectstorage.Storage,
	productRepo product.Repository,
) ImageGCUsecase {
	return &imageGCUsecase{
		tracer:      tf.Usecase(),
		clock:       clk,
		storage:     storage,
		productRepo: productRepo,
	}
}

func (u *imageGCUsecase) SweepOrphans(
	ctx context.Context, grace time.Duration, batchSize int32, dryRun bool,
) (ImageGCResult, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if grace <= 0 {
		grace = DefaultImageGCGrace
	}
	if batchSize <= 0 {
		batchSize = DefaultImageGCBatchSize
	}
	cutoff := u.clock.Now().Add(-grace)

	var (
		result ImageGCResult
		cursor string
	)
	for {
		page, err := u.sweepPage(ctx, cutoff, cursor, batchSize, dryRun)
		if err != nil {
			// 削除済みのオブジェクトは復元できない。累計を捨てると、実際に消えた件数が呼び手から見えなくなる。
			return result, err
		}
		result.Deleted += page.deleted
		result.Scanned += page.scanned

		if page.nextCursor == "" {
			return result, nil
		}
		cursor = page.nextCursor
	}
}

// sweepPage は、1 ページ分の列挙・参照照合・削除を実行します。
func (u *imageGCUsecase) sweepPage(
	ctx context.Context, cutoff time.Time, cursor string, batchSize int32, dryRun bool,
) (imageGCPageResult, error) {
	listed, err := u.storage.List(ctx, objectstorage.ListQuery{
		Prefix: imageKeyPrefix,
		Cursor: cursor,
		Limit:  batchSize,
	})
	if err != nil {
		return imageGCPageResult{}, err
	}

	page := imageGCPageResult{nextCursor: listed.NextCursor}
	candidates := agedImageKeys(listed.Objects, cutoff)
	if len(candidates) == 0 {
		return page, nil
	}
	page.scanned = int64(len(candidates))

	// 参照の照合に失敗したまま削除へ進むと、全オブジェクトが未参照に見えて生きている画像を消してしまう。
	// この経路だけは fail-open が許されないため、エラーはそのまま伝播して当ページの削除を行わない。
	referenced, err := u.productRepo.FilterExistingImagePaths(ctx, candidates)
	if err != nil {
		return imageGCPageResult{}, err
	}

	orphans := excludeKeys(candidates, referenced)
	if len(orphans) == 0 {
		return page, nil
	}
	if dryRun {
		page.deleted = int64(len(orphans))
		return page, nil
	}

	if err := u.storage.Delete(ctx, orphans); err != nil {
		return imageGCPageResult{}, err
	}
	page.deleted = int64(len(orphans))
	return page, nil
}

// agedImageKeys は、猶予期間を過ぎた商品画像オブジェクトのキーを返します。
// 接頭辞を再検査するのは、列挙の絞り込みを取りこぼした実装に当たっても
// 商品画像以外のオブジェクトを削除対象にしないためです。
func agedImageKeys(objects []objectstorage.Object, cutoff time.Time) []string {
	keys := make([]string, 0, len(objects))
	for _, o := range objects {
		if !strings.HasPrefix(o.Key, imageKeyPrefix) {
			continue
		}
		if !o.ModifiedAt.Before(cutoff) {
			continue
		}
		keys = append(keys, o.Key)
	}
	return keys
}

// excludeKeys は、keys から excluded に含まれるものを取り除いた並びを返します。
func excludeKeys(keys, excluded []string) []string {
	if len(excluded) == 0 {
		return keys
	}
	excludedSet := make(map[string]struct{}, len(excluded))
	for _, k := range excluded {
		excludedSet[k] = struct{}{}
	}
	remained := make([]string, 0, len(keys))
	for _, k := range keys {
		if _, ok := excludedSet[k]; !ok {
			remained = append(remained, k)
		}
	}
	return remained
}
