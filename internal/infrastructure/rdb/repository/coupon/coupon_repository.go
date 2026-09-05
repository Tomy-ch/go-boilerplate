// Package coupon は、クーポンリポジトリ（coupon.Repository）の RDB 実装を提供します。
package coupon

import (
	"context"

	"go-boilerplate/internal/domain/coupon"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"
)

type repository struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、クーポンリポジトリの RDB 実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) coupon.Repository {
	return &repository{
		db:     db,
		tracer: tf.Infra(),
	}
}

// CountByScopeTargetProductID は、指定商品を適用範囲の対象とするクーポンの発行枚数を取得します。
func (r *repository) CountByScopeTargetProductID(ctx context.Context, productID uuid.UUID) (int, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	count, err := db.SelectCouponCountByScopeTargetProductID(ctx, &productID)
	if err != nil {
		return 0, pgerror.NormalizeError(err)
	}

	return int(count), nil
}
