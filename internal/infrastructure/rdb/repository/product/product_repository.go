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
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/safecast"
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
// params.Ascending により昇順／降順を切り替え、CategoryID / StatusID / Keyword / price・quantity の範囲で絞り込みます。
func (r *repository) FindPublishedList(ctx context.Context, params product.ListParams) (product.Products, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	hasAfter := params.AfterPublishedAt != nil && params.AfterID != nil

	// keyset は Repository README の方針どおり、並び順ごとに first/after の固定クエリへ分ける。
	// 参照: internal/infrastructure/rdb/repository/README.md#keyset-pagination
	toRow := func(p gen.Products, statusName, categoryName string) productRow {
		return productRow{p: p, statusName: statusName, categoryName: categoryName}
	}

	switch {
	case params.Ascending && hasAfter:
		rows, err := db.ListPublishedProductsAscAfter(ctx, &gen.ListPublishedProductsAscAfterParams{
			CategoryID:       params.CategoryID,
			StatusID:         params.StatusID,
			Keyword:          params.Keyword,
			MinPrice:         ptr.Map(params.MinPrice, money.Price.Decimal),
			MaxPrice:         ptr.Map(params.MaxPrice, money.Price.Decimal),
			MinQuantity:      params.MinQuantity,
			MaxQuantity:      params.MaxQuantity,
			AfterPublishedAt: params.AfterPublishedAt,
			AfterID:          *params.AfterID,
			LimitParam:       params.Limit,
		})
		if err != nil {
			return nil, pgerror.NormalizeError(err)
		}
		return r.buildProducts(ctx, toProductRows(rows, func(row *gen.ListPublishedProductsAscAfterRow) productRow {
			return toRow(row.Products, row.StatusName, row.CategoryName)
		}))
	case params.Ascending:
		rows, err := db.ListPublishedProductsAscFirst(ctx, &gen.ListPublishedProductsAscFirstParams{
			CategoryID:  params.CategoryID,
			StatusID:    params.StatusID,
			Keyword:     params.Keyword,
			MinPrice:    ptr.Map(params.MinPrice, money.Price.Decimal),
			MaxPrice:    ptr.Map(params.MaxPrice, money.Price.Decimal),
			MinQuantity: params.MinQuantity,
			MaxQuantity: params.MaxQuantity,
			LimitParam:  params.Limit,
		})
		if err != nil {
			return nil, pgerror.NormalizeError(err)
		}
		return r.buildProducts(ctx, toProductRows(rows, func(row *gen.ListPublishedProductsAscFirstRow) productRow {
			return toRow(row.Products, row.StatusName, row.CategoryName)
		}))
	case hasAfter:
		rows, err := db.ListPublishedProductsDescAfter(ctx, &gen.ListPublishedProductsDescAfterParams{
			CategoryID:       params.CategoryID,
			StatusID:         params.StatusID,
			Keyword:          params.Keyword,
			MinPrice:         ptr.Map(params.MinPrice, money.Price.Decimal),
			MaxPrice:         ptr.Map(params.MaxPrice, money.Price.Decimal),
			MinQuantity:      params.MinQuantity,
			MaxQuantity:      params.MaxQuantity,
			AfterPublishedAt: params.AfterPublishedAt,
			AfterID:          *params.AfterID,
			LimitParam:       params.Limit,
		})
		if err != nil {
			return nil, pgerror.NormalizeError(err)
		}
		return r.buildProducts(ctx, toProductRows(rows, func(row *gen.ListPublishedProductsDescAfterRow) productRow {
			return toRow(row.Products, row.StatusName, row.CategoryName)
		}))
	default:
		rows, err := db.ListPublishedProductsDescFirst(ctx, &gen.ListPublishedProductsDescFirstParams{
			CategoryID:  params.CategoryID,
			StatusID:    params.StatusID,
			Keyword:     params.Keyword,
			MinPrice:    ptr.Map(params.MinPrice, money.Price.Decimal),
			MaxPrice:    ptr.Map(params.MaxPrice, money.Price.Decimal),
			MinQuantity: params.MinQuantity,
			MaxQuantity: params.MaxQuantity,
			LimitParam:  params.Limit,
		})
		if err != nil {
			return nil, pgerror.NormalizeError(err)
		}
		return r.buildProducts(ctx, toProductRows(rows, func(row *gen.ListPublishedProductsDescFirstRow) productRow {
			return toRow(row.Products, row.StatusName, row.CategoryName)
		}))
	}
}

// CountPublished は、公開済み商品のうち指定された検索条件に一致する件数を返します。
func (r *repository) CountPublished(ctx context.Context, params product.CountPublishedParams) (int64, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	count, err := db.CountPublishedProductsByFilter(ctx, &gen.CountPublishedProductsByFilterParams{
		CategoryID:  params.CategoryID,
		StatusID:    params.StatusID,
		MinPrice:    ptr.Map(params.MinPrice, money.Price.Decimal),
		MaxPrice:    ptr.Map(params.MaxPrice, money.Price.Decimal),
		MinQuantity: params.MinQuantity,
		MaxQuantity: params.MaxQuantity,
		Keyword:     params.Keyword,
	})
	if err != nil {
		return 0, pgerror.NormalizeError(err)
	}
	return count, nil
}

// FindAllLowStock は、在庫警告閾値以下の商品を在庫数と ID の昇順で最大 limit 件取得します。
// 在庫警告閾値が未設定の商品は除外し、公開状態では絞り込みません。
func (r *repository) FindAllLowStock(ctx context.Context, limit int32) (product.Products, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	rows, err := db.ListLowStockProducts(ctx, limit)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	return r.buildProducts(ctx, toProductRows(rows, func(row *gen.ListLowStockProductsRow) productRow {
		return productRow{p: row.Products, statusName: row.StatusName, categoryName: row.CategoryName}
	}))
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

	return r.buildProduct(ctx, productRow{p: row.Products, statusName: row.StatusName, categoryName: row.CategoryName})
}

// FindByID は、公開状態を問わず ID から単一商品を取得します。未存在は NotFound を返します。
// 公開日時の設定を更新対象にするため、未公開商品も返します。
func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*product.Product, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	row, err := db.GetProductByID(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	return r.buildProduct(ctx, productRow{p: row.Products, statusName: row.StatusName, categoryName: row.CategoryName})
}

// LockByID は、更新のため ID から公開状態を問わない単一商品を悲観ロックして取得します。
// 未存在は NotFound を返します。
func (r *repository) LockByID(ctx context.Context, id uuid.UUID) (*product.Product, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	row, err := db.GetProductByIDForUpdate(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	return r.buildProduct(ctx, productRow{p: row.Products, statusName: row.StatusName, categoryName: row.CategoryName})
}

// LockByIDs は、更新のため ID の集合から公開状態を問わない商品群を ID 昇順に悲観ロックして取得します。
// 不存在の ID は結果に現れないため、要素数は ids より少なくなり得ます。
func (r *repository) LockByIDs(ctx context.Context, ids []uuid.UUID) (product.Products, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	rows, err := db.ListProductsByIDsForUpdate(ctx, ids)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	return r.buildProducts(ctx, toProductRows(rows, func(row *gen.ListProductsByIDsForUpdateRow) productRow {
		return productRow{p: row.Products, statusName: row.StatusName, categoryName: row.CategoryName}
	}))
}

// UpdateStock は、p が保持するバージョンを条件に在庫数を更新し、採番後のバージョンを返します。
func (r *repository) UpdateStock(ctx context.Context, p *product.Product) (int, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	quantity, err := safecast.IntToInt32(p.Quantity())
	if err != nil {
		return 0, xerrors.Wrap(err, "invalid product quantity")
	}
	currentVersion, err := safecast.IntToInt32(p.Version())
	if err != nil {
		return 0, xerrors.Wrap(err, "invalid product version")
	}

	db := gen.New(driver.New(ctx, r.db))
	lockVersion, err := db.UpdateProductStock(ctx, &gen.UpdateProductStockParams{
		Quantity:       quantity,
		ID:             p.ID(),
		CurrentVersion: currentVersion,
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

// Create は、商品を新規登録します。p が保持する画像も併せて登録します。
func (r *repository) Create(ctx context.Context, p *product.Product) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	quantity, err := safecast.IntToInt32(p.Quantity())
	if err != nil {
		return xerrors.Wrap(err, "invalid product quantity")
	}
	stockWarningThreshold, err := safecast.IntPtrToInt32Ptr(p.StockWarningThreshold())
	if err != nil {
		return xerrors.Wrap(err, "invalid product stockWarningThreshold")
	}

	db := gen.New(driver.New(ctx, r.db))
	err = db.CreateProduct(ctx, &gen.CreateProductParams{
		ID:                    p.ID(),
		Name:                  p.Name(),
		Description:           p.Description(),
		Price:                 p.Price().Decimal(),
		Quantity:              quantity,
		StockWarningThreshold: stockWarningThreshold,
		StatusID:              p.Status().ID(),
		CategoryID:            p.Category().ID(),
		PublishedAt:           p.PublishedAt(),
	})
	if err != nil {
		return pgerror.NormalizeError(err)
	}

	return r.insertImages(ctx, db, p)
}

// ReplaceImages は、商品が現在参照している画像を p が保持する画像で置き換えます。
// 置き換え前の画像は論理削除として残ります。
//
// 同一商品への置換が直列化されるのは、先行する Update の条件付き UPDATE が商品行のロックを取るためです。
func (r *repository) ReplaceImages(ctx context.Context, p *product.Product) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	if err := db.SoftDeleteProductImages(ctx, p.ID()); err != nil {
		return pgerror.NormalizeError(err)
	}

	return r.insertImages(ctx, db, p)
}

// Update は、p が保持するバージョンを条件に商品を更新し、採番後のバージョンを返します。
// 画像は対象に含みません（置換は ReplaceImages が担います）。
func (r *repository) Update(ctx context.Context, p *product.Product) (int, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	quantity, err := safecast.IntToInt32(p.Quantity())
	if err != nil {
		return 0, xerrors.Wrap(err, "invalid product quantity")
	}
	stockWarningThreshold, err := safecast.IntPtrToInt32Ptr(p.StockWarningThreshold())
	if err != nil {
		return 0, xerrors.Wrap(err, "invalid product stockWarningThreshold")
	}
	currentVersion, err := safecast.IntToInt32(p.Version())
	if err != nil {
		return 0, xerrors.Wrap(err, "invalid product version")
	}

	db := gen.New(driver.New(ctx, r.db))
	lockVersion, err := db.UpdateProduct(ctx, &gen.UpdateProductParams{
		Name:                  p.Name(),
		Description:           p.Description(),
		Price:                 p.Price().Decimal(),
		Quantity:              quantity,
		StockWarningThreshold: stockWarningThreshold,
		StatusID:              p.Status().ID(),
		CategoryID:            p.Category().ID(),
		PublishedAt:           p.PublishedAt(),
		ID:                    p.ID(),
		CurrentVersion:        currentVersion,
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

// Count は、登録商品の総数と公開済み件数を返します。商品が 1 件もない場合はゼロ値を返します。
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

// FilterExistingImagePaths は、paths のうち商品が現在の画像として参照しているものを重複排除して返します。
// 未参照オブジェクトの回収で「消してよいか」を判定するための照会で、返らなかったパスが孤児にあたります。
func (r *repository) FilterExistingImagePaths(ctx context.Context, paths []string) ([]string, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	if len(paths) == 0 {
		return nil, nil
	}

	db := gen.New(driver.New(ctx, r.db))
	existing, err := db.ListExistingProductImagePaths(ctx, paths)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	return existing, nil
}

// insertImages は、p が保持する画像を登録します。
func (r *repository) insertImages(ctx context.Context, db *gen.Queries, p *product.Product) error {
	for _, img := range p.Images() {
		sortKey, err := safecast.IntToInt16(img.SortKey())
		if err != nil {
			return xerrors.Wrap(err, "invalid product image sortKey")
		}

		if err = db.CreateProductImage(ctx, &gen.CreateProductImageParams{
			ID:        img.ID(),
			ProductID: p.ID(),
			ImagePath: img.ImagePath(),
			SortKey:   sortKey,
		}); err != nil {
			return pgerror.NormalizeError(err)
		}
	}

	return nil
}

// findImagesByProductIDs は、商品 ID ごとの画像を表示順の昇順で返します。
// 置き換えで論理削除された画像は現在の参照ではないため、SQL 側で除いています。
func (r *repository) findImagesByProductIDs(
	ctx context.Context, ids []uuid.UUID,
) (map[uuid.UUID][]product.Image, error) {
	images := make(map[uuid.UUID][]product.Image, len(ids))
	if len(ids) == 0 {
		return images, nil
	}

	db := gen.New(driver.New(ctx, r.db))
	rows, err := db.ListProductImagesByProductIDs(ctx, ids)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	for _, row := range rows {
		img := row.ProductImages
		images[img.ProductID] = append(images[img.ProductID], product.NewImage(img.ID, product.ImageAttributes{
			ImagePath: img.ImagePath,
			SortKey:   int(img.SortKey),
		}))
	}

	return images, nil
}

// buildProducts は、商品行の集合を画像込みのドメインエンティティ列へ変換します。
// 画像は行数によらず 1 度の問い合わせでまとめて取得します。
func (r *repository) buildProducts(ctx context.Context, rows []productRow) (product.Products, error) {
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.p.ID
	}
	imagesByProductID, err := r.findImagesByProductIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	products := make(product.Products, len(rows))
	for i, row := range rows {
		p, perr := rowToProduct(row, imagesByProductID[row.p.ID])
		if perr != nil {
			return nil, perr
		}
		products[i] = p
	}

	return products, nil
}

// buildProduct は、単一の商品行を画像込みのドメインエンティティへ変換します。
func (r *repository) buildProduct(ctx context.Context, row productRow) (*product.Product, error) {
	products, err := r.buildProducts(ctx, []productRow{row})
	if err != nil {
		return nil, err
	}

	return products[0], nil
}

// rowToProduct は、sqlc が返す商品行（マスタ JOIN 込み）と画像をドメインエンティティへ変換します。
func rowToProduct(row productRow, images []product.Image) (*product.Product, error) {
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
		StockWarningThreshold: ptr.Map(row.p.StockWarningThreshold, func(v int32) int { return int(v) }),
		Status:                status,
		Category:              category,
		PublishedAt:           row.p.PublishedAt,
		Images:                images,
	}, int(row.p.LockVersion))
	if err != nil {
		return nil, pgerror.NormalizeReconstructError(err)
	}
	return entity, nil
}

// toProductRows は、クエリごとに異なる行型を共通の変換元へ揃えます。
func toProductRows[T any](rows []T, extract func(T) productRow) []productRow {
	converted := make([]productRow, len(rows))
	for i, row := range rows {
		converted[i] = extract(row)
	}
	return converted
}
