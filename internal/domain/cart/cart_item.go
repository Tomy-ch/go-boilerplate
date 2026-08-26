package cart

import (
	"time"

	"go-boilerplate/internal/domain/lexicon/money"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"
)

// CartItem は、カートの明細を表す値オブジェクトです。
// 明細はカート集約に属する部品であり単独では成立しないため、不変条件の判定は集約へ入る時点で行われます。
type CartItem struct {
	id            uuid.UUID
	productID     uuid.UUID
	quantity      int
	addedAt       time.Time
	lastSeenPrice *money.Price
}

// CartItemAttributes は、明細の組み立てに必要な属性一式です。id と productID は隣接する同じ
// uuid.UUID で、位置引数のままだと取り違えてもコンパイルも検証も通ってしまうため構造体で受けます。
type CartItemAttributes struct {
	// ProductID は、明細の対象商品です。
	ProductID uuid.UUID
	// Quantity は、明細の数量です。
	Quantity int
	// AddedAt は、明細が最初にカートへ入った時刻です。数量の置換では引き継がれます。
	AddedAt time.Time
	// LastSeenPrice は、最後に利用者へ提示した価格です。未提示は nil です。
	LastSeenPrice *money.Price
}

// NewCartItem は、明細を組み立てます。
// 集約の不変条件（同一商品の重複、明細数の上限）は明細 1 件では判定できないため、ここでは検証しません。
func NewCartItem(id uuid.UUID, attrs CartItemAttributes) CartItem {
	return CartItem{
		id:            id,
		productID:     attrs.ProductID,
		quantity:      attrs.Quantity,
		addedAt:       attrs.AddedAt,
		lastSeenPrice: ptr.Copy(attrs.LastSeenPrice),
	}
}

// ID は、明細 ID を返します。
func (i CartItem) ID() uuid.UUID { return i.id }

// ProductID は、商品 ID を返します。
func (i CartItem) ProductID() uuid.UUID { return i.productID }

// Quantity は、数量を返します。
func (i CartItem) Quantity() int { return i.quantity }

// AddedAt は、明細が最初にカートへ入った時刻を返します。
func (i CartItem) AddedAt() time.Time { return i.addedAt }

// LastSeenPrice は、最後に利用者へ提示した価格を返します。未提示の場合は nil です。
//
// これは比較の基準であって金額の根拠ではありません。請求額を拘束するのは購入明細のスナップショットだけです。
func (i CartItem) LastSeenPrice() *money.Price { return ptr.Copy(i.lastSeenPrice) }
