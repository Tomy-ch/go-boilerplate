// Package summary は、購入集計クエリサービス（query.PurchaseSummaryQueryService）の RDB 実装を提供します。
package summary

import (
	"context"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/purchase/query"
	"go-boilerplate/internal/usecase/tools/timewindow"
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
// 所有権は WHERE 述語（user_id 一致）で担保するため、他ユーザーの購入は集計に混入しません
// （本ファイルの他メソッドも同じ担保）。対象がない場合は空スライスを返します。
func (s *service) SummarizeByUserID(
	ctx context.Context, userID uuid.UUID, window timewindow.Window,
) ([]query.PurchaseStatusSummaryReadModel, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	rows, err := db.SummarizePurchasesByUserID(ctx, &gen.SummarizePurchasesByUserIDParams{
		UserID:        userID,
		OrderedAfter:  window.After(),
		OrderedBefore: window.Before(),
	})
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	results := make([]query.PurchaseStatusSummaryReadModel, len(rows))
	for i, row := range rows {
		results[i] = query.PurchaseStatusSummaryReadModel{
			StatusID:    row.StatusID,
			StatusCode:  int(row.StatusCode),
			StatusName:  row.StatusName,
			Count:       row.PurchaseCount,
			TotalAmount: row.TotalAmount,
		}
	}
	return results, nil
}

// SumItemsByUserID は、認証主体（userID）の購入明細の金額合計を価格スケールの decimal で返します。
// 所有権の担保は SummarizeByUserID を参照。対象がない場合はゼロ値を返します。
func (s *service) SumItemsByUserID(
	ctx context.Context, userID uuid.UUID, window timewindow.Window,
) (decimal.Decimal, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	total, err := db.SumPurchaseItemsByUserID(ctx, &gen.SumPurchaseItemsByUserIDParams{
		UserID:        userID,
		OrderedAfter:  window.After(),
		OrderedBefore: window.Before(),
	})
	if err != nil {
		return decimal.Decimal{}, pgerror.NormalizeError(err)
	}
	return total, nil
}

// SummarizeItemsByProductByUserID は、認証主体（userID）の購入明細を商品単位に集計して返します。
// カテゴリ名は商品カテゴリマスタとの一意な等結合で解決します。
// 所有権の担保は SummarizeByUserID を参照。対象がない場合は空スライスを返します。
func (s *service) SummarizeItemsByProductByUserID(
	ctx context.Context, userID uuid.UUID, window timewindow.Window,
) ([]query.PurchaseItemSummaryReadModel, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	rows, err := db.SummarizePurchaseItemsByProductByUserID(ctx, &gen.SummarizePurchaseItemsByProductByUserIDParams{
		UserID:        userID,
		OrderedAfter:  window.After(),
		OrderedBefore: window.Before(),
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
