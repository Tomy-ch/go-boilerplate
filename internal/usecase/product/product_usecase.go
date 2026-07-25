//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package product は、商品の参照・作成ユースケースと商品画像アップロードユースケースを提供します。
package product

import (
	"context"
	"time"

	"go-boilerplate/internal/apperror"
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
	// Ascending は、公開日時の昇順で取得する場合に true、降順の場合に false です。
	Ascending bool
}

// Usecase は、商品の参照ユースケースと画像アップロードユースケースを定義します。
type Usecase interface {
	// ListProducts は、公開済み商品を公開日時順（cursor ページネーション）で取得します。
	ListProducts(ctx context.Context, params ListProductsParams) (*ProductListView, error)
	// GetProduct は、ID から公開中の単一商品を取得します。未存在・非公開はいずれも NotFound を返します（存在秘匿）。
	GetProduct(ctx context.Context, id uuid.UUID) (ProductView, error)
	// UploadProductImage は、admin が商品画像をアップロードし、格納先のオブジェクトパスを返します。
	// 非 admin は 403、非対応形式は 415、サイズ超過は 413、空データは 422 を返します。
	UploadProductImage(ctx context.Context, authn *auth.Authn, params UploadProductImageParams) (ProductImageView, error)
	// CreateProduct は、admin が商品を作成し、作成した商品を返します。未認証は 401、非 admin は 403、
	// 負価格・負在庫・名称長超過などの業務不変条件違反は 422、status / category の不在は整合性異常として 500 を返します。
	CreateProduct(ctx context.Context, authn *auth.Authn, params CreateProductParams) (ProductView, error)
}

// usecase は、Usecase の実装です。
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

// New は、商品の参照・作成・画像アップロードユースケースを生成します。
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

// ListProducts は、公開済み商品を公開日時順（cursor ページネーション）で取得します。
func (u *usecase) ListProducts(ctx context.Context, params ListProductsParams) (*ProductListView, error) {
	if params.Cursor == nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "cursor must not be nil")
	}

	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	after, err := decodeProductCursor(params.Cursor)
	if err != nil {
		return nil, err
	}

	domainParams := product.ListParams{
		Limit:      params.Cursor.Limit32() + 1,
		Ascending:  params.Ascending,
		CategoryID: params.CategoryID,
		StatusID:   params.StatusID,
		Keyword:    params.Keyword,
	}
	if after != nil {
		publishedAt := after.publishedAt
		id := after.id
		domainParams.AfterPublishedAt = &publishedAt
		domainParams.AfterID = &id
	}

	products, err := u.repo.FindPublishedList(ctx, domainParams)
	if err != nil {
		return nil, err
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
		encoded := encodeProductCursor(products[len(products)-1])
		nextCursor = &encoded
	}

	return &ProductListView{Items: items, NextCursor: nextCursor}, nil
}

// GetProduct は、ID から公開中の単一商品を取得します。
// 未存在・非公開は Repository が NotFound を返すため、そのまま伝播して 404 に落とします（存在秘匿）。
func (u *usecase) GetProduct(ctx context.Context, id uuid.UUID) (ProductView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	p, err := u.repo.FindPublishedByID(ctx, id)
	if err != nil {
		return ProductView{}, err
	}

	return toProductView(p), nil
}

// toProductView は、商品エンティティを出力 DTO へ変換します。
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
	}
}
