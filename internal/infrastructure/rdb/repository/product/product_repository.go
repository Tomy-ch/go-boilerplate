// Package product は、商品リポジトリ（product.Repository）の RDB 実装を提供します。
package product

import (
	"context"
	"time"

	"go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"
)

type repository struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、product.Repository の RDB 実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) product.Repository {
	return &repository{
		db:     db,
		tracer: tf.Infra(),
	}
}

// FindPublishedList は、公開済み商品を (published_at, id) の keyset ページネーションで取得します。
// params.Ascending により昇順／降順を切り替え、CategoryID / StatusID / Keyword で絞り込みます。
func (r *repository) FindPublishedList(ctx context.Context, params product.ListParams) (product.Products, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	hasAfter := params.AfterPublishedAt != nil && params.AfterID != nil

	if params.Ascending {
		rows, err := db.ListPublishedProductsAsc(ctx, &gen.ListPublishedProductsAscParams{
			CategoryID:       params.CategoryID,
			StatusID:         params.StatusID,
			Keyword:          params.Keyword,
			HasAfter:         hasAfter,
			AfterPublishedAt: params.AfterPublishedAt,
			AfterID:          params.AfterID,
			LimitParam:       params.Limit,
		})
		if err != nil {
			return nil, pgerror.NormalizeError(err)
		}
		return rowsToProducts(rows, func(row *gen.ListPublishedProductsAscRow) gen.Products { return row.Products })
	}

	rows, err := db.ListPublishedProductsDesc(ctx, &gen.ListPublishedProductsDescParams{
		CategoryID:       params.CategoryID,
		StatusID:         params.StatusID,
		Keyword:          params.Keyword,
		HasAfter:         hasAfter,
		AfterPublishedAt: params.AfterPublishedAt,
		AfterID:          params.AfterID,
		LimitParam:       params.Limit,
	})
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	return rowsToProducts(rows, func(row *gen.ListPublishedProductsDescRow) gen.Products { return row.Products })
}

// FindPublishedByID は、ID から公開中（published_at 非 NULL）の単一商品を取得します。
// 非公開・未存在はいずれも SQL の該当なし（sql.ErrNoRows）に落ち、NotFound へ正規化して返します。
func (r *repository) FindPublishedByID(ctx context.Context, id uuid.UUID) (*product.Product, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	row, err := db.GetPublishedProductByID(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	return rowToProduct(row.Products)
}

// rowToProduct は、sqlc が返す Products 行をドメインエンティティへ変換します。
// 再構築時の検証失敗はデータ不整合として ErrInternal へ正規化します（422 / details にしない）。
func rowToProduct(p gen.Products) (*product.Product, error) {
	entity, err := product.New(
		p.ID,
		p.Name,
		p.Description,
		int(p.Price),
		int(p.Quantity),
		int32PtrToIntPtr(p.StockWarningThreshold),
		p.StatusID,
		p.CategoryID,
		ptr.Deref(p.PublishedAt, time.Time{}),
	)
	if err != nil {
		return nil, pgerror.NormalizeReconstructError(err)
	}
	return entity, nil
}

// rowsToProducts は、行スライスをドメインエンティティ列へ変換します。
func rowsToProducts[T any](rows []T, extract func(T) gen.Products) (product.Products, error) {
	products := make(product.Products, len(rows))
	for i, row := range rows {
		p, err := rowToProduct(extract(row))
		if err != nil {
			return nil, err
		}
		products[i] = p
	}
	return products, nil
}

// int32PtrToIntPtr は、sqlc の *int32 をドメインの *int へ変換します（nil はそのまま nil）。
func int32PtrToIntPtr(v *int32) *int {
	if v == nil {
		return nil
	}
	i := int(*v)
	return &i
}
