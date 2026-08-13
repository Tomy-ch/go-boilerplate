// Package productcategory は、商品カテゴリリポジトリ（category.Repository）の RDB 実装を提供します。
package productcategory

import (
	"context"

	"go-boilerplate/internal/domain/product/category"
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

// New は、category.Repository の RDB 実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) category.Repository {
	return &repository{
		db:     db,
		tracer: tf.Infra(),
	}
}

// FindAll は、全商品カテゴリエンティティを sortKey 昇順で取得します。
func (r *repository) FindAll(ctx context.Context) (category.Categories, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	rows, err := db.GetProductCategoryDomainAll(ctx)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	categories := make(category.Categories, len(rows))
	for i, row := range rows {
		categoryEntity, err := rowToProductCategory(row.ID, row.Name, row.Code, row.SortKey)
		if err != nil {
			return nil, err
		}
		categories[i] = categoryEntity
	}

	return categories, nil
}

// FindByID は、ID から単一の商品カテゴリエンティティを取得します。未存在は NotFound を返します。
func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*category.Category, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	row, err := db.GetProductCategoryByID(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	return rowToProductCategory(row.ID, row.Name, row.Code, row.SortKey)
}

// rowToProductCategory は、DB 行の値からドメインエンティティを再構築します。
func rowToProductCategory(id uuid.UUID, name string, code, sortKey int16) (*category.Category, error) {
	entity, err := category.New(id, category.Attributes{Name: name, Code: int(code), SortKey: int(sortKey)})
	if err != nil {
		return nil, pgerror.NormalizeReconstructError(err)
	}
	return entity, nil
}
