// Package coupon は、クーポンドメインを定義します。値引き（Discount）と適用範囲（Scope）という
// 直交する 2 つの値オブジェクトを持つ Coupon エンティティと、Repository インターフェースを提供します。
//
// 種別はマスタ表ではなくドメインが閉じた集合として持ち切ります。「定額か定率か」「全体かカテゴリか商品か」は
// 業務の語彙であってデータではなく、行として編集できることに意味がないためです（purchase.Status と同じ形）。
//
// 1 枚のクーポンは 1 つの値引きと 1 つの適用範囲を持ちます。複数枚の併用も、複数の適用範囲の合成も
// 表しません。
package coupon

import (
	"time"

	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// Coupons は、Coupon エンティティのスライス型です。
type Coupons []*Coupon

// Coupon は、クーポンを表すドメインエンティティです。
//
// 受給者は発行時に確定し、以後移りません。譲渡を表さないのは、クーポンが特定の利用者に生じた事情への
// 補償として発行されるためです。
type Coupon struct {
	id        uuid.UUID
	userID    uuid.UUID
	discount  Discount
	scope     Scope
	expiresAt time.Time
	usedAt    *time.Time
	issuedAt  time.Time
}

// Attributes は、クーポンの属性一式です。ExpiresAt と IssuedAt が同型のため構造体で受けます
// （基準は docs/rules.md の Function Signature Rules）。
type Attributes struct {
	// UserID は、受給者です。
	UserID uuid.UUID
	// Discount は、いくら引くかです。
	Discount Discount
	// Scope は、どの明細が対象かです。
	Scope Scope
	// ExpiresAt は、有効期限です。
	ExpiresAt time.Time
	// IssuedAt は、発行日時です。
	IssuedAt time.Time
}

// New は、クーポンエンティティの検証と生成を行います。生成直後は未使用です。
// id・UserID が未設定、Discount・Scope が未設定、ExpiresAt・IssuedAt がゼロ値の場合は検証エラーを返します。
// 有効期限が発行日時以前の場合も検証エラーです（発行した時点で使えないクーポンは発行の意味を持たないため）。
func New(id uuid.UUID, attrs Attributes) (*Coupon, error) {
	return newCoupon(id, attrs, nil)
}

// Reconstruct は、永続化済みのクーポンを再構築します。usedAt は使用済みなら使用日時、
// 未使用なら nil です。その他の検証は New と同一です。
func Reconstruct(id uuid.UUID, attrs Attributes, usedAt *time.Time) (*Coupon, error) {
	return newCoupon(id, attrs, usedAt)
}

// newCoupon は、生成・再構築に共通の検証を行いクーポンエンティティを構築します。
func newCoupon(id uuid.UUID, attrs Attributes, usedAt *time.Time) (*Coupon, error) {
	if id.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidID, "id is required")
	}
	if attrs.UserID.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidUserID, "userID is required")
	}
	if attrs.Discount.IsZero() {
		return nil, ErrInvalidDiscount
	}
	if attrs.Scope.IsZero() {
		return nil, ErrInvalidScope
	}
	if attrs.IssuedAt.IsZero() {
		return nil, xerrors.Wrap(ErrInvalidIssuedAt, "issuedAt is required")
	}
	if attrs.ExpiresAt.IsZero() {
		return nil, xerrors.Wrap(ErrInvalidExpiresAt, "expiresAt is required")
	}
	if !attrs.ExpiresAt.After(attrs.IssuedAt) {
		return nil, xerrors.Wrap(ErrInvalidExpiresAt, "expiresAt must be after issuedAt")
	}

	var used *time.Time
	if usedAt != nil {
		u := *usedAt
		used = &u
	}

	return &Coupon{
		id:        id,
		userID:    attrs.UserID,
		discount:  attrs.Discount,
		scope:     attrs.Scope,
		expiresAt: attrs.ExpiresAt,
		usedAt:    used,
		issuedAt:  attrs.IssuedAt,
	}, nil
}

// ID は、クーポン ID を返します。
func (c *Coupon) ID() uuid.UUID { return c.id }

// UserID は、受給者のユーザー ID を返します。
func (c *Coupon) UserID() uuid.UUID { return c.userID }

// Discount は、値引きを返します。
func (c *Coupon) Discount() Discount { return c.discount }

// Scope は、適用範囲を返します。
func (c *Coupon) Scope() Scope { return c.scope }

// ExpiresAt は、有効期限を返します。
func (c *Coupon) ExpiresAt() time.Time { return c.expiresAt }

// IssuedAt は、発行日時を返します。
func (c *Coupon) IssuedAt() time.Time { return c.issuedAt }

// UsedAt は、使用日時を返します。未使用の場合は nil です。
func (c *Coupon) UsedAt() *time.Time {
	if c.usedAt == nil {
		return nil
	}
	used := *c.usedAt
	return &used
}

// IsUsed は、クーポンが使用済みかどうかを返します。
func (c *Coupon) IsUsed() bool { return c.usedAt != nil }

// IsExpired は、渡された時点でクーポンが失効しているかどうかを返します。
//
// 「失効」の定義はこのメソッドが持ちます。失効を一括更新する機構は持たず、判定のたびに現在時刻と
// 突き合わせます（カートの期限切れと同じ形）。時刻はドメインの外から渡します。
func (c *Coupon) IsExpired(now time.Time) bool { return !now.Before(c.expiresAt) }
