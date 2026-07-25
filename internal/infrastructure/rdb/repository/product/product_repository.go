// Package product は、商品リポジトリ（product.Repository）の RDB 実装を提供します。
package product

import (
	"context"

	"go-boilerplate/internal/domain/kernel/money"
	"go-boilerplate/internal/domain/product"
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

// productRow は、商品行と固定参照マスタ JOIN で解決したステータス名・カテゴリ名をまとめた変換元です。
type productRow struct {
	p            gen.Products
	statusName   string
	categoryName string
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
		return rowsToProducts(rows, func(row *gen.ListPublishedProductsAscRow) productRow {
			return productRow{p: row.Products, statusName: row.StatusName, categoryName: row.CategoryName}
		})
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
	return rowsToProducts(rows, func(row *gen.ListPublishedProductsDescRow) productRow {
		return productRow{p: row.Products, statusName: row.StatusName, categoryName: row.CategoryName}
	})
}

// FindPublishedByID は、ID から公開中（published_at 非 NULL）の単一商品を取得します。
// 非公開・未存在はいずれも NotFound を返します（存在秘匿）。
func (r *repository) FindPublishedByID(ctx context.Context, id uuid.UUID) (*product.Product, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	row, err := db.GetPublishedProductByID(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	return rowToProduct(productRow{p: row.Products, statusName: row.StatusName, categoryName: row.CategoryName})
}

// Create は、商品を新規登録します。
func (r *repository) Create(ctx context.Context, p *product.Product) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	err := db.CreateProduct(ctx, &gen.CreateProductParams{
		ID:                    p.ID(),
		Name:                  p.Name(),
		Description:           p.Description(),
		Price:                 p.Price().Decimal(),
		Quantity:              int32(p.Quantity()), //nolint:gosec // G115: quantity は int32 の DB 列に収まる範囲で検証済み
		StockWarningThreshold: intPtrToInt32Ptr(p.StockWarningThreshold()),
		StatusID:              p.Status().ID(),
		CategoryID:            p.Category().ID(),
		PublishedAt:           p.PublishedAt(),
		ImagePath:             p.ImagePath(),
	})
	if err != nil {
		return pgerror.NormalizeError(err)
	}
	return nil
}

// intPtrToInt32Ptr は、ドメインの *int を sqlc の *int32 へ変換します（nil はそのまま nil）。
func intPtrToInt32Ptr(v *int) *int32 {
	if v == nil {
		return nil
	}
	//nolint:gosec // G115: stockWarningThreshold は int32 の DB 列に収まる範囲で検証済み
	i := int32(*v)
	return &i
}

// rowToProduct は、sqlc が返す商品行（マスタ JOIN 込み）をドメインエンティティへ変換します。
// 再構築時の検証失敗はデータ不整合として ErrInternal へ正規化します（422 / details にしない）。
func rowToProduct(row productRow) (*product.Product, error) {
	price, err := money.NewPrice(row.p.Price)
	if err != nil {
		return nil, pgerror.NormalizeReconstructError(err)
	}
	status, err := product.NewStatusRef(row.p.StatusID, row.statusName)
	if err != nil {
		return nil, pgerror.NormalizeReconstructError(err)
	}
	category, err := product.NewCategoryRef(row.p.CategoryID, row.categoryName)
	if err != nil {
		return nil, pgerror.NormalizeReconstructError(err)
	}

	entity, err := product.New(
		row.p.ID,
		row.p.Name,
		row.p.Description,
		price,
		int(row.p.Quantity),
		int32PtrToIntPtr(row.p.StockWarningThreshold),
		status,
		category,
		row.p.PublishedAt,
		row.p.ImagePath,
	)
	if err != nil {
		return nil, pgerror.NormalizeReconstructError(err)
	}
	return entity, nil
}

// rowsToProducts は、行スライスをドメインエンティティ列へ変換します。
func rowsToProducts[T any](rows []T, extract func(T) productRow) (product.Products, error) {
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
