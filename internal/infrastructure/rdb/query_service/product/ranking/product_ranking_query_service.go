// Package ranking は、商品売上ランキングのクエリサービス実装を提供します。
package ranking

import (
	"context"
	"time"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/internal/usecase/product/ranking/query"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/xerrors"
)

// rankingWindow30d は、period=30d の集計対象とする直近期間の長さです。
const rankingWindow30d = 30 * 24 * time.Hour

type service struct {
	db     driver.DatabaseDriver
	clk    clock.Clock
	tracer observability.LayerTracer
}

// New は、商品売上ランキングのクエリサービス実装を生成して返します。
func New(db driver.DatabaseDriver, clk clock.Clock, tf observability.TracerFactory) query.ProductRankingQueryService {
	return &service{
		db:     db,
		clk:    clk,
		tracer: tf.Infra(),
	}
}

// ListRanking は、購入明細を集計した商品売上ランキングを販売数量の降順で返します。
func (s *service) ListRanking(ctx context.Context, params query.RankingQueryParams) ([]query.RankingResult, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	filterByPeriod, orderedAfter := resolvePeriod(params.Period, s.clk.Now())

	limitCount, err := safecast.IntToInt32(params.Limit)
	if err != nil {
		return nil, xerrors.Wrap(err, "invalid ranking limit")
	}

	db := gen.New(driver.New(ctx, s.db))
	rows, err := db.ListProductRanking(ctx, &gen.ListProductRankingParams{
		FilterByPeriod: filterByPeriod,
		OrderedAfter:   orderedAfter,
		LimitCount:     limitCount,
	})
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	results := make([]query.RankingResult, len(rows))
	for i, row := range rows {
		results[i] = query.RankingResult{
			ProductID:    row.ProductID,
			Name:         row.Name,
			Price:        row.Price,
			PublishedAt:  row.PublishedAt,
			SoldQuantity: row.SoldQuantity,
		}
	}
	return results, nil
}

// resolvePeriod は、集計期間区分を SQL パラメータ（期間フィルタ有無と境界時刻）へ変換します。
// period=30d のときのみ now を基準に直近30日の境界を算出し、現在時刻への依存をインフラ層へ閉じ込めます。
func resolvePeriod(period query.Period, now time.Time) (bool, *time.Time) {
	if period != query.Period30d {
		return false, nil
	}
	after := now.Add(-rankingWindow30d)
	return true, &after
}
