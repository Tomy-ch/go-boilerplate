// Package dashboard は、admin ダッシュボード横断集計クエリサービス（query.DashboardQueryService）の RDB 実装を提供します。
package dashboard

import (
	"context"
	"time"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/internal/usecase/dashboard/query"
)

type service struct {
	db     driver.DatabaseDriver
	clk    clock.Clock
	loc    *time.Location
	tracer observability.LayerTracer
}

// New は、admin ダッシュボード横断集計クエリサービスの RDB 実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	clk clock.Clock,
	loc *time.Location,
	tf observability.TracerFactory,
) query.DashboardQueryService {
	return &service{
		db:     db,
		clk:    clk,
		loc:    loc,
		tracer: tf.Infra(),
	}
}

// SummarizeSales は、指定期間に注文された購入の売上合計と件数を返します。
// キャンセル済みの購入は WHERE 述語（canceled_at IS NULL）で除外し、未払いの購入は含めます。
func (s *service) SummarizeSales(ctx context.Context, period query.Period) (query.SalesResult, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	after, before := resolveWindow(period, s.clk.Now(), s.loc)

	db := gen.New(driver.New(ctx, s.db))
	row, err := db.SummarizeDashboardSales(ctx, &gen.SummarizeDashboardSalesParams{
		OrderedAfter:  after,
		OrderedBefore: before,
	})
	if err != nil {
		return query.SalesResult{}, pgerror.NormalizeError(err)
	}

	return query.SalesResult{Amount: row.SalesAmount, Count: row.SalesCount}, nil
}

// CountPurchasesByStatus は、指定期間に注文された購入のステータス別件数を返します。
// ステータス名は purchase_statuses との一意な等結合で解決するため、別途の名称解決は不要です。
func (s *service) CountPurchasesByStatus(
	ctx context.Context, period query.Period,
) ([]query.PurchaseStatusCountResult, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	after, before := resolveWindow(period, s.clk.Now(), s.loc)

	db := gen.New(driver.New(ctx, s.db))
	rows, err := db.CountDashboardPurchasesByStatus(ctx, &gen.CountDashboardPurchasesByStatusParams{
		OrderedAfter:  after,
		OrderedBefore: before,
	})
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	results := make([]query.PurchaseStatusCountResult, len(rows))
	for i, row := range rows {
		results[i] = query.PurchaseStatusCountResult{
			StatusID:   row.StatusID,
			StatusName: row.StatusName,
			Count:      row.PurchaseCount,
		}
	}
	return results, nil
}

// resolveWindow は、集計対象期間の指定を SQL パラメータの半開区間 [after, before) へ変換します。
// 現在時刻とタイムゾーンへの依存をインフラ層へ閉じ込め、暦日の境界は loc で解釈します。
// loc は設定のタイムゾーンから構築された値であり、実行環境の time.Local には依存しません
// （コンテナの既定は UTC のため、依存させると設定と異なる暦日で集計してしまいます）。
// 区分が未知の場合は today として扱います（ユースケース側で正規化済みのため通常は到達しません）。
func resolveWindow(period query.Period, now time.Time, loc *time.Location) (time.Time, time.Time) {
	// 呼出側の変換有無に依存せず loc 基準の暦日を得るため、ここで現在時刻を loc へ移す。
	// period.From / To は利用者が指定した暦日そのものを表すため、同じ変換をしてはならない
	// （UTC より西のロケーションでは前日へずれる）。
	now = now.In(loc)

	switch period.Kind {
	case query.PeriodMonth:
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(0, 1, 0)
	case query.PeriodRange:
		return startOfDay(period.From, loc), startOfDay(period.To, loc).AddDate(0, 0, 1)
	case query.PeriodToday:
		start := startOfDay(now, loc)
		return start, start.AddDate(0, 0, 1)
	default:
		start := startOfDay(now, loc)
		return start, start.AddDate(0, 0, 1)
	}
}

// startOfDay は、t が表す年月日の開始時刻を loc のゾーンで返します。年月日は t 自身のロケーションで解釈し、
// loc は返り値のゾーンとしてのみ用います（t を loc へ変換し直しません）。
func startOfDay(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}
