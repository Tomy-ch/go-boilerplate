//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package product は、商品の参照ユースケースを提供します。
package product

import (
	"context"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// ProductView は、商品 1 件分のユースケース出力 DTO です。price は USD セント単位の整数です。
type ProductView struct {
	ID                    uuid.UUID
	Name                  string
	Description           *string
	Price                 int
	Quantity              int
	StockWarningThreshold *int
	StatusID              uuid.UUID
	CategoryID            uuid.UUID
	PublishedAt           time.Time
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

// Usecase は、商品の参照ユースケースを定義します。
type Usecase interface {
	// ListProducts は、公開済み商品を公開日時順（cursor ページネーション）で取得します。
	ListProducts(ctx context.Context, params ListProductsParams) (*ProductListView, error)
}

// usecase は、Usecase の実装です。
type usecase struct {
	tracer observability.LayerTracer
	repo   product.Repository
}

// New は、商品の参照ユースケースを生成します。
func New(repo product.Repository, tf observability.TracerFactory) Usecase {
	return &usecase{
		tracer: tf.Usecase(),
		repo:   repo,
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

// toProductView は、商品エンティティを出力 DTO へ変換します。
func toProductView(p *product.Product) ProductView {
	return ProductView{
		ID:                    p.ID(),
		Name:                  p.Name(),
		Description:           p.Description(),
		Price:                 p.Price(),
		Quantity:              p.Quantity(),
		StockWarningThreshold: p.StockWarningThreshold(),
		StatusID:              p.StatusID(),
		CategoryID:            p.CategoryID(),
		PublishedAt:           p.PublishedAt(),
	}
}
