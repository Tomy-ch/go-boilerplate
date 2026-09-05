// Package product は、商品画像クエリサービス（query.ProductImageQueryService）の RDB 実装を提供します。
package product

import (
	"context"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/product/query"
)

type service struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、商品画像クエリサービスの RDB 実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) query.ProductImageQueryService {
	return &service{
		db:     db,
		tracer: tf.Infra(),
	}
}

func (s *service) FilterExistingImagePaths(ctx context.Context, paths []string) ([]string, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	if len(paths) == 0 {
		return nil, nil
	}

	db := gen.New(driver.New(ctx, s.db))

	existing, err := db.ListExistingProductImagePaths(ctx, paths)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	return existing, nil
}
