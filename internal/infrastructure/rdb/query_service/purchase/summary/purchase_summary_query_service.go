// Package summary は、購入集計クエリサービス（query.PurchaseSummaryQueryService）の RDB 実装を提供します。
package summary

import (
	"context"
	"time"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/purchase/period"
	"go-boilerplate/internal/usecase/purchase/query"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/uuid"
)

type service struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、購入集計クエリサービスの RDB 実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) query.PurchaseSummaryQueryService {
	return &service{
		db:     db,
		tracer: tf.Infra(),
	}
}

// SummarizeByUserID は、認証主体（userID）の購入をステータス単位に集計して購入ステータスマスタの表示順で返します。
// 所有権は WHERE 述語（user_id 一致）で担保するため、他ユーザーの購入は集計に混入しません。
// 対象がない場合は空スライスを返します。
func (s *service) SummarizeByUserID(
	ctx context.Context, userID uuid.UUID, window period.Window,
) ([]query.PurchaseStatusSummaryReadModel, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	filter, after, before := bounds(window)

	db := gen.New(driver.New(ctx, s.db))
	rows, err := db.SummarizePurchasesByUserID(ctx, &gen.SummarizePurchasesByUserIDParams{
		UserID:         userID,
		FilterByPeriod: filter,
		OrderedAfter:   after,
		OrderedBefore:  before,
	})
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	results := make([]query.PurchaseStatusSummaryReadModel, len(rows))
	for i, row := range rows {
		results[i] = query.PurchaseStatusSummaryReadModel{
			StatusID:    row.StatusID,
			StatusName:  row.StatusName,
			Count:       row.PurchaseCount,
			TotalAmount: row.TotalAmount,
		}
	}
	return results, nil
}

// SumItemsByUserID は、認証主体（userID）の購入明細の金額合計を価格スケールの decimal で返します。
// 所有権は WHERE 述語（user_id 一致）で担保します。対象がない場合はゼロ値を返します。
func (s *service) SumItemsByUserID(
	ctx context.Context, userID uuid.UUID, window period.Window,
) (decimal.Decimal, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	filter, after, before := bounds(window)

	db := gen.New(driver.New(ctx, s.db))
	total, err := db.SumPurchaseItemsByUserID(ctx, &gen.SumPurchaseItemsByUserIDParams{
		UserID:         userID,
		FilterByPeriod: filter,
		OrderedAfter:   after,
		OrderedBefore:  before,
	})
	if err != nil {
		return decimal.Decimal{}, pgerror.NormalizeError(err)
	}
	return total, nil
}

// SummarizeItemsByProductByUserID は、認証主体（userID）の購入明細を商品単位に集計して返します。
// カテゴリ名は商品カテゴリマスタとの一意な等結合で解決するため、別途の名称解決は不要です。
// 所有権は WHERE 述語（user_id 一致）で担保します。対象がない場合は空スライスを返します。
func (s *service) SummarizeItemsByProductByUserID(
	ctx context.Context, userID uuid.UUID, window period.Window,
) ([]query.PurchaseItemSummaryReadModel, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	filter, after, before := bounds(window)

	db := gen.New(driver.New(ctx, s.db))
	rows, err := db.SummarizePurchaseItemsByProductByUserID(ctx, &gen.SummarizePurchaseItemsByProductByUserIDParams{
		UserID:         userID,
		FilterByPeriod: filter,
		OrderedAfter:   after,
		OrderedBefore:  before,
	})
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	results := make([]query.PurchaseItemSummaryReadModel, len(rows))
	for i, row := range rows {
		results[i] = query.PurchaseItemSummaryReadModel{
			CategoryName: row.CategoryName,
			ProductID:    row.ProductID,
			ProductName:  row.ProductName,
			ItemsTotal:   row.ItemsTotal,
		}
	}
	return results, nil
}

// bounds は、解決済みの対象期間を SQL パラメータの組（絞り込むか / 下限 / 上限）へ変換します。
// 絞り込まない指定では境界を NULL のまま渡し、フラグ側だけで述語を無効化します。
func bounds(window period.Window) (bool, *time.Time, *time.Time) {
	if !window.Filtered() {
		return false, nil, nil
	}
	after, before := window.Bounds()
	return true, &after, &before
}
