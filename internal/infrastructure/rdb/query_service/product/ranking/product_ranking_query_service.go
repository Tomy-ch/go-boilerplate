// Package ranking は、商品売上ランキングのクエリサービス実装を提供します。
package ranking

import (
	"context"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/product/ranking/query"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/xerrors"
)

type service struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、商品売上ランキングのクエリサービス実装を生成して返します。
func New(db driver.DatabaseDriver, tf observability.TracerFactory) query.ProductRankingQueryService {
	return &service{
		db:     db,
		tracer: tf.Infra(),
	}
}

// ListRanking は、購入明細を集計した商品売上ランキングを販売数量の降順で返します。
func (s *service) ListRanking(ctx context.Context, params query.RankingQueryParams) ([]query.RankingResult, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	limitCount, err := safecast.IntToInt32(params.Limit)
	if err != nil {
		return nil, xerrors.Wrap(err, "invalid ranking limit")
	}

	db := gen.New(driver.New(ctx, s.db))
	rows, err := db.ListProductRanking(ctx, &gen.ListProductRankingParams{
		OrderedAfter:  params.Window.After(),
		OrderedBefore: params.Window.Before(),
		LimitCount:    limitCount,
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
