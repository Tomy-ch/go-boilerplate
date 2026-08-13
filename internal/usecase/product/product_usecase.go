//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package product は、商品の参照・作成ユースケースと商品画像アップロードユースケースを提供します。
package product

import (
	"context"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/lexicon/money"
	"go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/domain/product/category"
	"go-boilerplate/internal/domain/product/status"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	"go-boilerplate/internal/usecase/boundary/objectstorage"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

const maxProductPriceFilterLength = 40

// errUnpublishedInPublishedRead は、公開中として取得した読み取りに未公開の商品が混じっていた場合のエラーです。
// 絞り込みを実行する SQL と、公開中を定義する Product.IsPublished が食い違ったことを意味します。
var errUnpublishedInPublishedRead = xerrors.Wrap(apperror.ErrInternal, "unpublished product in published read")

// ProductView は、商品 1 件分のユースケース出力 DTO です。Price はサブセント精度を保持する価格スケールの十進量です。
// ステータス・カテゴリは商品集約の一部として ID と名称を保持します（画面側での再解決は不要です）。
type ProductView struct {
	ID                    uuid.UUID
	Name                  string
	Description           *string
	Price                 decimal.Decimal
	Quantity              int
	StockWarningThreshold *int
	StatusID              uuid.UUID
	StatusName            string
	CategoryID            uuid.UUID
	CategoryName          string
	PublishedAt           *time.Time
	ImagePath             *string
	// Version は、楽観ロックのバージョンです。部分更新の要求へそのまま渡すことで競合を検出できます。
	Version int
}

// ProductListView は、公開商品一覧（cursor ページネーション）の取得結果を表します。
type ProductListView struct {
	// Items は、現在ページの商品一覧です。
	Items []ProductView
	// NextCursor は、次ページ取得用の不透明カーソルです。最終ページの場合は nil です。
	NextCursor *string
}

// ListProductsParams は、公開商品一覧取得の入力パラメータです。
type ListProductsParams struct {
	// Cursor は、cursor ページネーションの取得件数と境界を表します。
	Cursor *paging.Cursor
	// CategoryID は、商品カテゴリ ID による絞り込みです。nil の場合は絞り込みません。
	CategoryID *uuid.UUID
	// StatusID は、商品ステータス ID による絞り込みです。nil の場合は絞り込みません。
	StatusID *uuid.UUID
	// Keyword は、商品名・説明への部分一致検索キーワードです。nil の場合は絞り込みません。
	Keyword *string
	// MinPrice / MaxPrice は、価格の包含下限／包含上限を表す十進文字列です。nil の側は制限しません。
	MinPrice *string
	MaxPrice *string
	// MinQuantity / MaxQuantity は、在庫数の包含下限／包含上限です。nil の側は制限しません。
	MinQuantity *int32
	MaxQuantity *int32
	// Ascending は、公開日時の昇順で取得する場合に true、降順の場合に false です。
	Ascending bool
}

type productListRange struct {
	minPrice    *money.Price
	maxPrice    *money.Price
	minQuantity *int32
	maxQuantity *int32
}

// Usecase は、商品に関するアプリケーションユースケースを定義します。
type Usecase interface {
	// ListProducts は、公開済み商品を公開日時順（cursor ページネーション）で取得します。
	ListProducts(ctx context.Context, params ListProductsParams) (ProductListView, error)
	// GetProduct は、ID から公開中の単一商品を取得します。未存在・非公開はいずれも NotFound を返します（存在秘匿）。
	GetProduct(ctx context.Context, id uuid.UUID) (ProductView, error)
	// UploadProductImage は、admin が商品画像をアップロードし、格納先のオブジェクトパスを返します。
	// 非 admin は 403、非対応形式は 415、サイズ超過は 413、空データは 422 を返します。
	UploadProductImage(ctx context.Context, authn *auth.Authn, params UploadProductImageParams) (ProductImageView, error)
	// CreateProduct は、admin が商品を作成し、作成した商品を返します。未認証は 401、非 admin は 403、
	// 負価格・負在庫・名称長超過などの業務不変条件違反は 422、status / category の不在は整合性異常として 500 を返します。
	CreateProduct(ctx context.Context, authn *auth.Authn, params CreateProductParams) (ProductView, error)
	// UpdateProduct は、admin が商品の属性を部分更新し、更新後の商品を返します。未認証は 401、非 admin は 403、
	// 未存在は 404、読み込み後に他者が更新していた場合は 409、業務不変条件違反は 422 を返します。
	UpdateProduct(ctx context.Context, authn *auth.Authn, id uuid.UUID, params UpdateProductParams) (ProductView, error)
	// UpdateProductStock は、admin が商品の在庫を増減し、更新後の商品を返します。未認証は 401、非 admin は 403、
	// 未存在は 404、取得後に他者が更新していた場合は 409、増減後の在庫が保持できる範囲を外れる場合は 422 を返します。
	UpdateProductStock(
		ctx context.Context, authn *auth.Authn, id uuid.UUID, params UpdateProductStockParams,
	) (ProductView, error)
	// ListLowStockProducts は、admin が在庫警告閾値以下まで在庫が減った商品を在庫の少ない順に取得します。
	// 未認証は 401、非 admin は 403 を返します。
	ListLowStockProducts(
		ctx context.Context, authn *auth.Authn, params ListLowStockProductsParams,
	) (ProductLowStockListView, error)
}

type usecase struct {
	tracer         observability.LayerTracer
	txm            tx.Manager
	repo           product.Repository
	categoryRepo   category.Repository
	statusRepo     status.Repository
	storage        objectstorage.Storage
	authorizer     authz.Authorizer
	maxUploadBytes int64
}

// New は、商品に関するユースケース実装を生成します。
func New(
	txm tx.Manager,
	repo product.Repository,
	categoryRepo category.Repository,
	statusRepo status.Repository,
	storage objectstorage.Storage,
	authorizer authz.Authorizer,
	maxUploadBytes int64,
	tf observability.TracerFactory,
) Usecase {
	return &usecase{
		tracer:         tf.Usecase(),
		txm:            txm,
		repo:           repo,
		categoryRepo:   categoryRepo,
		statusRepo:     statusRepo,
		storage:        storage,
		authorizer:     authorizer,
		maxUploadBytes: maxUploadBytes,
	}
}

func parseProductPriceFilter(name, value string) (*money.Price, error) {
	if len(value) > maxProductPriceFilterLength {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, name+" is too long")
	}
	parsed, err := decimal.Parse(value)
	if err != nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, name+" must be a non-negative decimal")
	}
	price, err := money.NewPrice(parsed)
	if err != nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, name+" must be a non-negative decimal")
	}
	return &price, nil
}

func parseProductListRange(params ListProductsParams) (productListRange, error) {
	result := productListRange{minQuantity: params.MinQuantity, maxQuantity: params.MaxQuantity}
	if params.MinPrice != nil {
		minPrice, err := parseProductPriceFilter("minPrice", *params.MinPrice)
		if err != nil {
			return productListRange{}, err
		}
		result.minPrice = minPrice
	}
	if params.MaxPrice != nil {
		maxPrice, err := parseProductPriceFilter("maxPrice", *params.MaxPrice)
		if err != nil {
			return productListRange{}, err
		}
		result.maxPrice = maxPrice
	}
	if result.minPrice != nil && result.maxPrice != nil &&
		result.minPrice.Decimal().Cmp(result.maxPrice.Decimal()) > 0 {
		return productListRange{}, xerrors.Wrap(apperror.ErrInvalidArgument, "minPrice must not exceed maxPrice")
	}
	if result.minQuantity != nil && *result.minQuantity < 0 {
		return productListRange{}, xerrors.Wrap(apperror.ErrInvalidArgument, "minQuantity must be non-negative")
	}
	if result.maxQuantity != nil && *result.maxQuantity < 0 {
		return productListRange{}, xerrors.Wrap(apperror.ErrInvalidArgument, "maxQuantity must be non-negative")
	}
	if result.minQuantity != nil && result.maxQuantity != nil && *result.minQuantity > *result.maxQuantity {
		return productListRange{}, xerrors.Wrap(apperror.ErrInvalidArgument, "minQuantity must not exceed maxQuantity")
	}
	return result, nil
}

func (u *usecase) ListProducts(ctx context.Context, params ListProductsParams) (ProductListView, error) {
	if params.Cursor == nil {
		return ProductListView{}, xerrors.Wrap(apperror.ErrInvalidArgument, "cursor must not be nil")
	}

	rangeFilter, err := parseProductListRange(params)
	if err != nil {
		return ProductListView{}, err
	}

	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	after, err := decodeProductCursor(params.Cursor)
	if err != nil {
		return ProductListView{}, err
	}

	domainParams := product.ListParams{
		Limit:       params.Cursor.Limit32() + 1,
		Ascending:   params.Ascending,
		CategoryID:  params.CategoryID,
		StatusID:    params.StatusID,
		Keyword:     params.Keyword,
		MinPrice:    rangeFilter.minPrice,
		MaxPrice:    rangeFilter.maxPrice,
		MinQuantity: rangeFilter.minQuantity,
		MaxQuantity: rangeFilter.maxQuantity,
	}
	if after != nil {
		publishedAt := after.publishedAt
		id := after.id
		domainParams.AfterPublishedAt = &publishedAt
		domainParams.AfterID = &id
	}

	products, err := u.repo.FindPublishedList(ctx, domainParams)
	if err != nil {
		return ProductListView{}, err
	}
	if err := ensurePublished(products); err != nil {
		return ProductListView{}, err
	}

	limit := params.Cursor.Limit()
	hasNext := len(products) > limit
	if hasNext {
		products = products[:limit]
	}

	items := make([]ProductView, len(products))
	for i, p := range products {
		items[i] = toProductView(p)
	}

	var nextCursor *string
	if hasNext && len(products) > 0 {
		// len 判定は防御的な安全弁。hasNext は len > limit なので limit >= 1 の下では冗長だが、
		// limit の下限保証は paging.NewCursor 依存であり、ゼロ値 Cursor 混入時の products[-1] panic を防ぐ。
		encoded := encodeProductCursor(products[len(products)-1])
		nextCursor = &encoded
	}

	return ProductListView{Items: items, NextCursor: nextCursor}, nil
}

// GetProduct は、存在秘匿を Repository が返す NotFound に委ね、usecase 側では公開判定を再実装しません。
// Repository のエラーをそのまま伝播させることで、未存在と非公開が 404 として区別不能に保たれます。
func (u *usecase) GetProduct(ctx context.Context, id uuid.UUID) (ProductView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	p, err := u.repo.FindPublishedByID(ctx, id)
	if err != nil {
		return ProductView{}, err
	}
	if err := ensurePublished(product.Products{p}); err != nil {
		return ProductView{}, err
	}

	return toProductView(p), nil
}

// ensurePublished は、Repository が公開中として返した商品がドメイン定義でも公開中であることを確かめます。
// 背景: internal/usecase/README.md § Verifying infrastructure against the domain.
func ensurePublished(products product.Products) error {
	for _, p := range products {
		if !p.IsPublished() {
			return xerrors.Wrap(errUnpublishedInPublishedRead, p.ID().String())
		}
	}
	return nil
}

func toProductView(p *product.Product) ProductView {
	return ProductView{
		ID:                    p.ID(),
		Name:                  p.Name(),
		Description:           p.Description(),
		Price:                 p.Price().Decimal(),
		Quantity:              p.Quantity(),
		StockWarningThreshold: p.StockWarningThreshold(),
		StatusID:              p.Status().ID(),
		StatusName:            p.Status().Name(),
		CategoryID:            p.Category().ID(),
		CategoryName:          p.Category().Name(),
		PublishedAt:           p.PublishedAt(),
		ImagePath:             p.ImagePath(),
		Version:               p.Version(),
	}
}
