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
// version は並行更新による上書き（lost update）を防ぐ楽観ロックのバージョンです。
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
	version               int
}

// Attributes は、商品の属性一式です。生成・再構築・更新のいずれの入口も同じ集合を受け取ります。
// Description と ImagePath のように同型の属性が並ぶため、位置引数ではなくフィールド名で与えることで、
// 属性の取り違えをコンパイルエラーとして検出できるようにしています。
type Attributes struct {
	Name                  string
	Description           *string
	Price                 money.Price
	Quantity              int
	StockWarningThreshold *int
	Status                StatusRef
	Category              CategoryRef
	PublishedAt           *time.Time
	ImagePath             *string
}

// New は、商品エンティティの検証と生成を行います。Price は非負の money.Price（非負検証は Price VO が担保）、
// Quantity は 0 以上、StockWarningThreshold は指定時 0 以上である必要があります。
// PublishedAt は nil（未公開）を許容し、ImagePath は無検証で保持します。
// id が nil、Name が長さ制約外、Status / Category がゼロ値の場合はそれぞれ検証エラーを返します。
// 生成直後のバージョンは initialVersion です。
func New(id uuid.UUID, attrs Attributes) (*Product, error) {
	return newProduct(id, attrs, initialVersion)
}

// Reconstruct は、永続化済みの商品を再構築します。
// version は永続化されている楽観ロックのバージョンで、initialVersion 未満の場合は検証エラーを返します。
// その他の検証は New と同一です。
func Reconstruct(id uuid.UUID, attrs Attributes, version int) (*Product, error) {
	return newProduct(id, attrs, version)
}

// newProduct は、生成・再構築に共通の検証を行い商品エンティティを構築します。
func newProduct(id uuid.UUID, attrs Attributes, version int) (*Product, error) {
	if id.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidID, "id is required")
	}
	if err := validateAttributes(attrs); err != nil {
		return nil, err
	}
	if version < initialVersion {
		return nil, xerrors.Wrap(
			ErrInvalidVersion,
			fmt.Sprintf("version must be %d or greater, got %d", initialVersion, version),
		)
	}

	return &Product{
		id:                    id,
		name:                  attrs.Name,
		description:           ptr.Copy(attrs.Description),
		price:                 attrs.Price,
		quantity:              attrs.Quantity,
		stockWarningThreshold: ptr.Copy(attrs.StockWarningThreshold),
		status:                attrs.Status,
		category:              attrs.Category,
		publishedAt:           ptr.Copy(attrs.PublishedAt),
		imagePath:             ptr.Copy(attrs.ImagePath),
		version:               version,
	}, nil
}

// validateAttributes は、商品属性の不変条件を検証します。生成時と更新時で同一の条件を課します。
func validateAttributes(attrs Attributes) error {
	if ok, msg := stringkit.ValidateInRange(attrs.Name, minNameLength, maxNameLength); !ok {
		return xerrors.Wrap(ErrInvalidName, msg)
	}
	if attrs.Quantity < minQuantity {
		return xerrors.Wrap(
			ErrInvalidQuantity,
			fmt.Sprintf("quantity must be %d or greater, got %d", minQuantity, attrs.Quantity),
		)
	}
	if attrs.StockWarningThreshold != nil && *attrs.StockWarningThreshold < minThreshold {
		return xerrors.Wrap(
			ErrInvalidStockWarningThreshold,
			fmt.Sprintf("stockWarningThreshold must be %d or greater, got %d", minThreshold, *attrs.StockWarningThreshold),
		)
	}
	if attrs.Status.id.IsNil() {
		return xerrors.Wrap(ErrInvalidStatusID, "status is required")
	}
	if attrs.Category.id.IsNil() {
		return xerrors.Wrap(ErrInvalidCategoryID, "category is required")
	}
	return nil
}

// Update は、商品の属性を更新します。生成時と同一の不変条件を課し、違反する場合はエンティティを
// 変更せずに検証エラーを返します。attrs は部分更新を解決した後の確定値であり、据え置く属性には現在値が渡されます。
// バージョンは永続化の成否に依存するためここでは進めません（採番は Repository の条件付き更新が行います）。
func (p *Product) Update(attrs Attributes) error {
	if err := validateAttributes(attrs); err != nil {
		return err
	}

	p.name = attrs.Name
	p.description = ptr.Copy(attrs.Description)
	p.price = attrs.Price
	p.quantity = attrs.Quantity
	p.stockWarningThreshold = ptr.Copy(attrs.StockWarningThreshold)
	p.status = attrs.Status
	p.category = attrs.Category
	p.publishedAt = ptr.Copy(attrs.PublishedAt)
	p.imagePath = ptr.Copy(attrs.ImagePath)

	return nil
}

// EnsureVersion は、更新要求が指すバージョンが現在のバージョンと一致することを確認します。
// 一致しない場合は、読み込み後に他者が更新したものとして ErrVersionConflict を返します。
func (p *Product) EnsureVersion(expected int) error {
	if p.version != expected {
		return xerrors.Wrap(
			ErrVersionConflict,
			fmt.Sprintf("expected version %d, but current version is %d", expected, p.version),
		)
	}
	return nil
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

// Version は、楽観ロックのバージョンを返します。
func (p *Product) Version() int { return p.version }
