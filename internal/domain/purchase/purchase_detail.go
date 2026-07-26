package purchase

import (
	"time"

	"go-boilerplate/pkg/uuid"
)

// Detail は、購入 1 件の詳細読み取りモデルです。ステータス名は購入ステータスマスタで
// 解決済みで、書き込み集約 Purchase とは別型です（read 側・CQRS）。明細は購入明細（PurchaseDetail）の
// スライスで、金額はすべて USD セント単位の整数、CanceledAt は未キャンセルなら nil です。
type Detail struct {
	// ID は、購入 ID です。
	ID uuid.UUID
	// Code は、購入コード（UUIDv7 文字列・一意）です。
	Code string
	// UserID は、購入したユーザーの ID です。
	UserID uuid.UUID
	// StatusID は、購入ステータス ID（購入ステータスマスタで解決済み）です。
	StatusID uuid.UUID
	// StatusName は、購入ステータスの名称（購入ステータスマスタで解決済み）です。
	StatusName string
	// SubtotalAmount は、小計（USD セント）です。
	SubtotalAmount int
	// TaxAmount は、税額（USD セント）です。
	TaxAmount int
	// ShippingFee は、送料（USD セント）です。
	ShippingFee int
	// TotalAmount は、合計（USD セント）です。
	TotalAmount int
	// Details は、購入明細のスライスです。
	Details []PurchaseDetail
	// OrderedAt は、注文日時です。
	OrderedAt time.Time
	// PaidAt は、支払い日時です。未支払いの場合は nil です。
	PaidAt *time.Time
	// CanceledAt は、キャンセル日時です。未キャンセルの場合は nil です。
	CanceledAt *time.Time
}
