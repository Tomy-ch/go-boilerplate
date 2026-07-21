// Package product は、商品ドメインを定義します。名称・価格・在庫などの不変条件を持つ Product エンティティと、
// 公開商品の cursor 一覧取得を含む Repository インターフェースを提供します。price は USD セント単位の整数です。
package product

import (
	"fmt"
	"time"

	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/stringkit"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// Products は、Product エンティティのスライス型です。
type Products []*Product

// Product は、商品を表すドメインエンティティです。price は USD セント単位の整数で保持します。
type Product struct {
	id                    uuid.UUID
	name                  string
	description           *string
	price                 int
	quantity              int
	stockWarningThreshold *int
	statusID              uuid.UUID
	categoryID            uuid.UUID
	publishedAt           time.Time
}

// New は、商品エンティティの検証と生成を行います。
// price / quantity は 0 以上、stockWarningThreshold は指定時 0 以上である必要があります。
// id / statusID / categoryID が nil、name が長さ制約外、publishedAt がゼロ値の場合はそれぞれ検証エラーを返します。
func New(
	id uuid.UUID,
	name string,
	description *string,
	price int,
	quantity int,
	stockWarningThreshold *int,
	statusID uuid.UUID,
	categoryID uuid.UUID,
	publishedAt time.Time,
) (*Product, error) {
	if id.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidID, "id is required")
	}
	if ok, msg := stringkit.ValidateInRange(name, minNameLength, maxNameLength); !ok {
		return nil, xerrors.Wrap(ErrInvalidName, msg)
	}
	if price < minPrice {
		return nil, xerrors.Wrap(ErrInvalidPrice, fmt.Sprintf("price must be %d or greater, got %d", minPrice, price))
	}
	if quantity < minQuantity {
		return nil, xerrors.Wrap(ErrInvalidQuantity, fmt.Sprintf("quantity must be %d or greater, got %d", minQuantity, quantity))
	}
	if stockWarningThreshold != nil && *stockWarningThreshold < minThreshold {
		return nil, xerrors.Wrap(
			ErrInvalidStockWarningThreshold,
			fmt.Sprintf("stockWarningThreshold must be %d or greater, got %d", minThreshold, *stockWarningThreshold),
		)
	}
	if statusID.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidStatusID, "statusID is required")
	}
	if categoryID.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidCategoryID, "categoryID is required")
	}
	if publishedAt.IsZero() {
		return nil, xerrors.Wrap(ErrInvalidPublishedAt, "publishedAt is required")
	}

	return &Product{
		id:                    id,
		name:                  name,
		description:           ptr.Copy(description),
		price:                 price,
		quantity:              quantity,
		stockWarningThreshold: ptr.Copy(stockWarningThreshold),
		statusID:              statusID,
		categoryID:            categoryID,
		publishedAt:           publishedAt,
	}, nil
}

// ID は、商品の ID を返します。
func (p *Product) ID() uuid.UUID { return p.id }

// Name は、商品名を返します。
func (p *Product) Name() string { return p.name }

// Description は、商品説明を返します。未設定の場合は nil です。
func (p *Product) Description() *string { return ptr.Copy(p.description) }

// Price は、価格（USD セント単位の整数）を返します。
func (p *Product) Price() int { return p.price }

// Quantity は、在庫数を返します。
func (p *Product) Quantity() int { return p.quantity }

// StockWarningThreshold は、在庫警告閾値を返します。未設定の場合は nil です。
func (p *Product) StockWarningThreshold() *int { return ptr.Copy(p.stockWarningThreshold) }

// StatusID は、商品ステータス ID を返します。
func (p *Product) StatusID() uuid.UUID { return p.statusID }

// CategoryID は、商品カテゴリ ID を返します。
func (p *Product) CategoryID() uuid.UUID { return p.categoryID }

// PublishedAt は、公開日時を返します。
func (p *Product) PublishedAt() time.Time { return p.publishedAt }
