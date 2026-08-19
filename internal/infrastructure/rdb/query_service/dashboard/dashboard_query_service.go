// Package dashboard は、admin ダッシュボード横断集計クエリサービス（query.DashboardQueryService）の RDB 実装を提供します。
package dashboard

import (
	"context"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/dashboard/query"
)

type service struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、admin ダッシュボード横断集計クエリサービスの RDB 実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) query.DashboardQueryService {
	return &service{
		db:     db,
		tracer: tf.Infra(),
	}
}

// SummarizeSales は、指定期間に注文された購入の売上合計と件数を返します。
// キャンセル済みの購入は WHERE 述語（canceled_at IS NULL）で除外し、未払いの購入は含めます。
func (s *service) SummarizeSales(ctx context.Context, window query.Window) (query.SalesResult, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	row, err := db.SummarizeDashboardSales(ctx, &gen.SummarizeDashboardSalesParams{
		OrderedAfter:  window.After,
		OrderedBefore: window.Before,
	})
	if err != nil {
		return query.SalesResult{}, pgerror.NormalizeError(err)
	}

	return query.SalesResult{Amount: row.SalesAmount, Count: row.SalesCount}, nil
}

// CountPurchasesByStatus は、指定期間に注文された購入のステータス別件数を返します。
// ステータス名は purchase_statuses との一意な等結合で解決します。
func (s *service) CountPurchasesByStatus(
	ctx context.Context, window query.Window,
) ([]query.PurchaseStatusCountResult, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))
	rows, err := db.CountDashboardPurchasesByStatus(ctx, &gen.CountDashboardPurchasesByStatusParams{
		OrderedAfter:  window.After,
		OrderedBefore: window.Before,
	})
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	results := make([]query.PurchaseStatusCountResult, len(rows))
	for i, row := range rows {
		results[i] = query.PurchaseStatusCountResult{
			StatusID:   row.StatusID,
			StatusCode: int(row.StatusCode),
			StatusName: row.StatusName,
			Count:      row.PurchaseCount,
		}
	}
	return results, nil
}
