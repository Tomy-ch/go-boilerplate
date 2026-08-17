// Package product は、商品ドメインを定義します。名称・価格・在庫などの不変条件を持つ Product エンティティと、
// 公開商品の cursor 一覧取得を含む Repository インターフェースを提供します。price はサブセント精度を保持する
// 価格スケール（Decimal）の値オブジェクト money.Price で保持します。
package product

import (
	"fmt"
	"slices"
	"time"

	"go-boilerplate/internal/domain/lexicon/money"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/stringkit"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// Products は、Product エンティティのスライス型です。
type Products []*Product

// Product は、商品を表すドメインエンティティです。price はサブセント精度を保持する money.Price で保持します。
// status / category はそれぞれ ID と名称を持つ参照で、呼び出し側での名称の別解決は不要です。
// publishedAt は未公開の場合 nil、images は画像未設定の場合は空で、表示順の昇順で保持します。
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
	images                []Image
	version               int
}

// Attributes は、商品の属性一式です。生成・再構築・更新のいずれの入口も同じ集合を受け取ります。
// ポインタのフィールドは未設定を nil で表し、不変条件は受け取り側が検証します。
type Attributes struct {
	Name                  string
	Description           *string
	Price                 money.Price
	Quantity              int
	StockWarningThreshold *int
	Status                StatusRef
	Category              CategoryRef
	PublishedAt           *time.Time
	Images                []Image
}

// New は、商品エンティティの検証と生成を行います。Price は非負の money.Price（非負検証は Price VO が担保）、
// Quantity と StockWarningThreshold（指定時）は、いずれも在庫が保持できる範囲に収まる必要があります。
// PublishedAt は nil（未公開）を許容し、Images は空（画像未設定）を許容します。
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
		images:                sortImagesByDisplaySort(attrs.Images),
		version:               version,
	}, nil
}

// validateAttributes は、商品属性の不変条件を検証します。生成時と更新時で同一の条件を課します。
func validateAttributes(attrs Attributes) error {
	if ok, msg := stringkit.ValidateInRange(attrs.Name, minNameLength, maxNameLength); !ok {
		return xerrors.Wrap(ErrInvalidName, msg)
	}
	if err := validateQuantity(int64(attrs.Quantity)); err != nil {
		return err
	}
	if err := validateStockWarningThreshold(attrs.StockWarningThreshold); err != nil {
		return err
	}
	if attrs.Status.id.IsNil() {
		return xerrors.Wrap(ErrInvalidStatusID, "status is required")
	}
	if attrs.Category.id.IsNil() {
		return xerrors.Wrap(ErrInvalidCategoryID, "category is required")
	}
	return validateImages(attrs.Images)
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
	p.images = sortImagesByDisplaySort(attrs.Images)

	return nil
}

// AdjustStock は、在庫数を delta の分だけ増減します。delta は正で補充、負で差し引きを表します。
// 増減後の在庫が負になる場合は、在庫を変更せずに検証エラーを返します。
// バージョンは永続化の成否に依存するためここでは進めません（採番は Repository の条件付き更新が行います）。
func (p *Product) AdjustStock(delta int) error {
	// 増減の途中結果は在庫の表現範囲を超えうるため、検証を通すまでは広い幅で保持します。
	adjusted := int64(p.quantity) + int64(delta)
	if err := validateQuantity(adjusted); err != nil {
		return err
	}

	p.quantity = int(adjusted)

	return nil
}

// validateQuantity は、在庫数が保持できる範囲に収まることを検証します。
// 生成・更新・在庫の増減のいずれの変更点でも同一の条件を課します。
func validateQuantity(quantity int64) error {
	if quantity < minQuantity || quantity > maxQuantity {
		return xerrors.Wrap(
			ErrInvalidQuantity,
			fmt.Sprintf("quantity must be between %d and %d, got %d", minQuantity, maxQuantity, quantity),
		)
	}
	return nil
}

// validateStockWarningThreshold は、在庫警告閾値が保持できる範囲に収まることを検証します。
// nil は閾値の未設定を表すため検証を通します。
func validateStockWarningThreshold(threshold *int) error {
	if threshold == nil {
		return nil
	}
	if *threshold < minThreshold || *threshold > maxThreshold {
		return xerrors.Wrap(
			ErrInvalidStockWarningThreshold,
			fmt.Sprintf("stockWarningThreshold must be between %d and %d, got %d", minThreshold, maxThreshold, *threshold),
		)
	}
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

// Images は、商品画像を表示順の昇順で返します。画像未設定の場合は空です。
func (p *Product) Images() []Image { return slices.Clone(p.images) }

// Version は、楽観ロックのバージョンを返します。
func (p *Product) Version() int { return p.version }

// IsPublished は、商品が公開中かどうかを返します。公開中とは公開日時が設定されていることを指します。
//
// 「公開中」の定義は [IsPublished] が持ちます。読み取り経路の絞り込みは SQL が実行しますが、それは
// この定義の実行形であって定義ではありません。
func (p *Product) IsPublished() bool { return IsPublished(p.publishedAt) }

// IsPublished は、公開日時から商品が公開中かどうかを判定します。
//
// 「公開中」の定義はここ 1 箇所にあります。集約を再構築しない読み取り——集計や射影を返す経路——にも
// 同じ定義を当てられるよう、エンティティのメソッドとは別に値に対する形でも公開します。
func IsPublished(publishedAt *time.Time) bool { return publishedAt != nil }

// IsLowStock は、商品の在庫が補充を要する水準まで減っているかどうかを返します。
// 在庫警告閾値が未設定の商品は警告対象を持たないため、常に false です。
//
// 「在庫僅少」の定義はこのメソッドが持ちます。IsPublished と同じく、SQL の絞り込みは実行形です。
func (p *Product) IsLowStock() bool {
	if p.stockWarningThreshold == nil {
		return false
	}
	return p.quantity <= *p.stockWarningThreshold
}
