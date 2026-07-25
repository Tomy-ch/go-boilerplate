// Package product は、商品ドメインを定義します。名称・価格・在庫などの不変条件を持つ Product エンティティと、
// 公開商品の cursor 一覧取得を含む Repository インターフェースを提供します。price はサブセント精度を保持する
// 価格スケール（Decimal）の値オブジェクト money.Price で保持します。
package product

import (
	"fmt"
	"time"

	"go-boilerplate/internal/domain/kernel/money"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/stringkit"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// Products は、Product エンティティのスライス型です。
type Products []*Product

// Product は、商品を表すドメインエンティティです。price はサブセント精度を保持する money.Price で保持します。
// status / category はそれぞれ ID と名称を持つ参照で、呼び出し側での名称の別解決は不要です。
// publishedAt は未公開の場合 nil、imagePath は画像未設定の場合 nil です。
type Product struct {
	id                    uuid.UUID
	name                  string
	description           *string
	price                 money.Price
	quantity              int
	stockWarningThreshold *int
	status                StatusRef
	category              CategoryRef
	publishedAt           *time.Time
	imagePath             *string
}

// New は、商品エンティティの検証と生成を行います。price は非負の money.Price（非負検証は Price VO が担保）、
// quantity は 0 以上、stockWarningThreshold は指定時 0 以上である必要があります。
// publishedAt は nil（未公開）を許容し、imagePath は無検証で保持します。
// id が nil、name が長さ制約外、status / category がゼロ値の場合はそれぞれ検証エラーを返します。
func New(
	id uuid.UUID,
	name string,
	description *string,
	price money.Price,
	quantity int,
	stockWarningThreshold *int,
	status StatusRef,
	category CategoryRef,
	publishedAt *time.Time,
	imagePath *string,
) (*Product, error) {
	if id.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidID, "id is required")
	}
	if ok, msg := stringkit.ValidateInRange(name, minNameLength, maxNameLength); !ok {
		return nil, xerrors.Wrap(ErrInvalidName, msg)
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
	if status.id.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidStatusID, "status is required")
	}
	if category.id.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidCategoryID, "category is required")
	}

	return &Product{
		id:                    id,
		name:                  name,
		description:           ptr.Copy(description),
		price:                 price,
		quantity:              quantity,
		stockWarningThreshold: ptr.Copy(stockWarningThreshold),
		status:                status,
		category:              category,
		publishedAt:           ptr.Copy(publishedAt),
		imagePath:             ptr.Copy(imagePath),
	}, nil
}

// ID は、商品の ID を返します。
func (p *Product) ID() uuid.UUID { return p.id }

// Name は、商品名を返します。
func (p *Product) Name() string { return p.name }

// Description は、商品説明を返します。未設定の場合は nil です。
func (p *Product) Description() *string { return ptr.Copy(p.description) }

// Price は、価格（サブセント精度を保持する money.Price）を返します。
func (p *Product) Price() money.Price { return p.price }

// Quantity は、在庫数を返します。
func (p *Product) Quantity() int { return p.quantity }

// StockWarningThreshold は、在庫警告閾値を返します。未設定の場合は nil です。
func (p *Product) StockWarningThreshold() *int { return ptr.Copy(p.stockWarningThreshold) }

// Status は、商品ステータス参照（ID + 名称）を返します。
func (p *Product) Status() StatusRef { return p.status }

// Category は、商品カテゴリ参照（ID + 名称）を返します。
func (p *Product) Category() CategoryRef { return p.category }

// PublishedAt は、公開日時を返します。未公開の場合は nil です。
func (p *Product) PublishedAt() *time.Time { return ptr.Copy(p.publishedAt) }

// ImagePath は、画像パスを返します。未設定の場合は nil です。
func (p *Product) ImagePath() *string { return ptr.Copy(p.imagePath) }
