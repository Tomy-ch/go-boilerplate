// Package product は、商品リポジトリ（product.Repository）の RDB 実装を提供します。
package product

import (
	"context"

	"go-boilerplate/internal/domain/lexicon/money"
	"go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
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

	// keyset は並び順ごとに「先頭ページ用」「カーソル以降用」の 2 本の固定クエリへ分ける。
	// 1 本へ畳んでオプショナル述語にすると、実行計画が入力に依存する単一のステートメントになる。
	toRow := func(p gen.Products, statusName, categoryName string) productRow {
		return productRow{p: p, statusName: statusName, categoryName: categoryName}
	}

	switch {
	case params.Ascending && hasAfter:
		rows, err := db.ListPublishedProductsAscAfter(ctx, &gen.ListPublishedProductsAscAfterParams{
			CategoryID:       params.CategoryID,
			StatusID:         params.StatusID,
			Keyword:          params.Keyword,
			AfterPublishedAt: params.AfterPublishedAt,
			AfterID:          *params.AfterID,
			LimitParam:       params.Limit,
		})
		if err != nil {
			return nil, pgerror.NormalizeError(err)
		}
		return rowsToProducts(rows, func(row *gen.ListPublishedProductsAscAfterRow) productRow {
			return toRow(row.Products, row.StatusName, row.CategoryName)
		})
	case params.Ascending:
		rows, err := db.ListPublishedProductsAscFirst(ctx, &gen.ListPublishedProductsAscFirstParams{
			CategoryID: params.CategoryID,
			StatusID:   params.StatusID,
			Keyword:    params.Keyword,
			LimitParam: params.Limit,
		})
		if err != nil {
			return nil, pgerror.NormalizeError(err)
		}
		return rowsToProducts(rows, func(row *gen.ListPublishedProductsAscFirstRow) productRow {
			return toRow(row.Products, row.StatusName, row.CategoryName)
		})
	case hasAfter:
		rows, err := db.ListPublishedProductsDescAfter(ctx, &gen.ListPublishedProductsDescAfterParams{
			CategoryID:       params.CategoryID,
			StatusID:         params.StatusID,
			Keyword:          params.Keyword,
			AfterPublishedAt: params.AfterPublishedAt,
			AfterID:          *params.AfterID,
			LimitParam:       params.Limit,
		})
		if err != nil {
			return nil, pgerror.NormalizeError(err)
		}
		return rowsToProducts(rows, func(row *gen.ListPublishedProductsDescAfterRow) productRow {
			return toRow(row.Products, row.StatusName, row.CategoryName)
		})
	default:
		rows, err := db.ListPublishedProductsDescFirst(ctx, &gen.ListPublishedProductsDescFirstParams{
			CategoryID: params.CategoryID,
			StatusID:   params.StatusID,
			Keyword:    params.Keyword,
			LimitParam: params.Limit,
		})
		if err != nil {
			return nil, pgerror.NormalizeError(err)
		}
		return rowsToProducts(rows, func(row *gen.ListPublishedProductsDescFirstRow) productRow {
			return toRow(row.Products, row.StatusName, row.CategoryName)
		})
	}
}

// FindAllLowStock は、ListLowStockProducts で quantity <= stock_warning_threshold の商品を
// (quantity, id) の昇順で最大 limit 件取得します。stock_warning_threshold が NULL の行は除外し、
// published_at では絞りません。
func (r *repository) FindAllLowStock(ctx context.Context, limit int32) (product.Products, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	rows, err := db.ListLowStockProducts(ctx, limit)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	return rowsToProducts(rows, func(row *gen.ListLowStockProductsRow) productRow {
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

// FindByID は、GetProductByID で published_at を絞らずに単一商品を取得します
// （公開中のみを返す FindPublishedByID との対で、未公開商品も返します）。未存在は NotFound を返します。
func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*product.Product, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	row, err := db.GetProductByID(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	return rowToProduct(productRow{p: row.Products, statusName: row.StatusName, categoryName: row.CategoryName})
}

// LockByID は、GetProductByIDForUpdate で対象行を悲観ロック（FOR UPDATE）したうえで取得します。
// 未存在は NotFound を返します。
func (r *repository) LockByID(ctx context.Context, id uuid.UUID) (*product.Product, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	row, err := db.GetProductByIDForUpdate(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	return rowToProduct(productRow{p: row.Products, statusName: row.StatusName, categoryName: row.CategoryName})
}

// LockByIDs は、ListProductsByIDsForUpdate で対象行を ID 昇順に悲観ロック（FOR UPDATE）したうえで取得します。
// 不存在の ID は結果に現れないため、要素数は ids より少なくなり得ます。
func (r *repository) LockByIDs(ctx context.Context, ids []uuid.UUID) (product.Products, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	rows, err := db.ListProductsByIDsForUpdate(ctx, ids)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	return rowsToProducts(rows, func(row *gen.ListProductsByIDsForUpdateRow) productRow {
		return productRow{p: row.Products, statusName: row.StatusName, categoryName: row.CategoryName}
	})
}

// UpdateStock は、p が保持するバージョンを条件に在庫数を更新し、採番後のバージョンを返します。
func (r *repository) UpdateStock(ctx context.Context, p *product.Product) (int, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	lockVersion, err := db.UpdateProductStock(ctx, &gen.UpdateProductStockParams{
		Quantity: int32(p.Quantity()), //nolint:gosec // G115: quantity は int32 の DB 列に収まる範囲で検証済み
		ID:       p.ID(),
		//nolint:gosec // G115: version は int32 の DB 列由来であり範囲に収まります
		CurrentVersion: int32(p.Version()),
	})
	if err != nil {
		// 対象行なしは、取得後に他トランザクションが更新しバージョンが進んだことを意味します
		// （存在は同一トランザクション内の取得で確認済みです）。
		// 行ロックを取っている経路では起こりませんが、ロックなしで呼ばれた場合に在庫を上書きしないための
		// 二重防御として衝突を返します。
		if pgerror.IsNoRows(err) {
			return 0, xerrors.Wrap(product.ErrVersionConflict, "product was updated by another transaction")
		}
		return 0, pgerror.NormalizeError(err)
	}

	return int(lockVersion), nil
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

// Update は、p が保持するバージョンを条件に商品を更新し、採番後のバージョンを返します。
func (r *repository) Update(ctx context.Context, p *product.Product) (int, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	lockVersion, err := db.UpdateProduct(ctx, &gen.UpdateProductParams{
		Name:                  p.Name(),
		Description:           p.Description(),
		Price:                 p.Price().Decimal(),
		Quantity:              int32(p.Quantity()), //nolint:gosec // G115: quantity は int32 の DB 列に収まる範囲で検証済み
		StockWarningThreshold: intPtrToInt32Ptr(p.StockWarningThreshold()),
		StatusID:              p.Status().ID(),
		CategoryID:            p.Category().ID(),
		PublishedAt:           p.PublishedAt(),
		ImagePath:             p.ImagePath(),
		ID:                    p.ID(),
		//nolint:gosec // G115: version は int32 の DB 列由来であり範囲に収まります
		CurrentVersion: int32(p.Version()),
	})
	if err != nil {
		// 対象行なしは、読み込み後に他トランザクションが更新しバージョンが進んだことを意味します
		// （存在は同一トランザクション内の読み込みで確認済みです）。
		// tx.Manager が透過リトライする一時障害（serialization_failure）と異なり同じ内容の再送では
		// 解消しないため、リトライ対象と混同されないよう衝突として返します。
		if pgerror.IsNoRows(err) {
			return 0, xerrors.Wrap(product.ErrVersionConflict, "product was updated by another transaction")
		}
		return 0, pgerror.NormalizeError(err)
	}

	return int(lockVersion), nil
}

// Count は、products の COUNT 集計から登録商品の総数と、published_at が非 NULL の公開済み件数を返します。
func (r *repository) Count(ctx context.Context) (product.Counts, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	row, err := db.CountProducts(ctx)
	if err != nil {
		return product.Counts{}, pgerror.NormalizeError(err)
	}

	return product.Counts{Total: row.TotalCount, Published: row.PublishedCount}, nil
}

// FilterExistingImagePaths は、paths のうち products が実際に参照しているものを重複排除して返します。
// 未参照オブジェクトの回収で「消してよいか」を判定するための照会で、返らなかったパスが孤児にあたります。
func (r *repository) FilterExistingImagePaths(ctx context.Context, paths []string) ([]string, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	if len(paths) == 0 {
		return nil, nil
	}

	db := gen.New(driver.New(ctx, r.db))
	rows, err := db.ListExistingProductImagePaths(ctx, paths)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	existing := make([]string, 0, len(rows))
	for _, p := range rows {
		// image_path は NULL 許容だが、NULL はどの要素とも一致しないため結果には現れない。
		// それでも nil を空文字として拾うと、参照されていないパスを参照済みと誤判定して孤児を残すため取り除く。
		if p == nil {
			continue
		}
		existing = append(existing, *p)
	}
	return existing, nil
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

	entity, err := product.Reconstruct(row.p.ID, product.Attributes{
		Name:                  row.p.Name,
		Description:           row.p.Description,
		Price:                 price,
		Quantity:              int(row.p.Quantity),
		StockWarningThreshold: int32PtrToIntPtr(row.p.StockWarningThreshold),
		Status:                status,
		Category:              category,
		PublishedAt:           row.p.PublishedAt,
		ImagePath:             row.p.ImagePath,
	}, int(row.p.LockVersion))
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
