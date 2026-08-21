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

// ListQuantityRanking は、購入明細を集計した販売数量ランキングを降順で返します。
func (s *service) ListQuantityRanking(
	ctx context.Context, params query.RankingQueryParams,
) ([]query.QuantityRankingResult, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	limitCount, err := safecast.IntToInt32(params.Limit)
	if err != nil {
		return nil, xerrors.Wrap(err, "invalid ranking limit")
	}

	db := gen.New(driver.New(ctx, s.db))
	rows, err := db.ListProductQuantityRanking(ctx, &gen.ListProductQuantityRankingParams{
		OrderedAfter:  params.Window.After(),
		OrderedBefore: params.Window.Before(),
		LimitCount:    limitCount,
	})
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	results := make([]query.QuantityRankingResult, len(rows))
	for i, row := range rows {
		results[i] = query.QuantityRankingResult{
			ProductID:    row.ProductID,
			Name:         row.Name,
			Price:        row.Price,
			PublishedAt:  row.PublishedAt,
			SoldQuantity: row.SoldQuantity,
		}
	}
	return results, nil
}

// ListAmountRanking は、購入明細を集計した売上金額ランキングを降順で返します。
func (s *service) ListAmountRanking(
	ctx context.Context, params query.RankingQueryParams,
) ([]query.AmountRankingResult, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	limitCount, err := safecast.IntToInt32(params.Limit)
	if err != nil {
		return nil, xerrors.Wrap(err, "invalid ranking limit")
	}

	db := gen.New(driver.New(ctx, s.db))
	rows, err := db.ListProductAmountRanking(ctx, &gen.ListProductAmountRankingParams{
		OrderedAfter:  params.Window.After(),
		OrderedBefore: params.Window.Before(),
		LimitCount:    limitCount,
	})
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	results := make([]query.AmountRankingResult, len(rows))
	for i, row := range rows {
		results[i] = query.AmountRankingResult{
			ProductID:   row.ProductID,
			Name:        row.Name,
			Price:       row.Price,
			PublishedAt: row.PublishedAt,
			SalesAmount: row.SalesAmount,
		}
	}
	return results, nil
}
