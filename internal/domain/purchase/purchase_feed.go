package purchase

import (
	"time"

	"go-boilerplate/pkg/uuid"
)

// FeedItem は、購入履歴一覧の 1 件分の読み取りモデルです。ステータス名は購入ステータスマスタとの
// 結合で解決済みで、書き込み集約 Purchase とは別型です（一覧は概要のみで明細を含みません）。
type FeedItem struct {
	// Code は、購入コード（UUIDv7 文字列・一意）です。
	Code string
	// TotalAmount は、合計金額（小計 + 税額 + 送料）です。USD セント単位の整数です。
	TotalAmount int
	// StatusID は、購入ステータス ID（購入ステータスマスタとの結合で解決済み）です。
	StatusID uuid.UUID
	// StatusName は、購入ステータスの名称（購入ステータスマスタで解決済み）です。
	StatusName string
	// OrderedAt は、注文日時です。keyset の主ソートキーです。
	OrderedAt time.Time
	// ID は、購入 ID です。keyset のタイブレークキーです。
	ID uuid.UUID
}

// ListFeedParams は、購入履歴フィード（keyset ページネーション）の取得条件です。
// AfterOrderedAt / AfterID は直前ページ末尾行の値で、先頭ページでは nil です
// （不透明カーソルの符号化・復号は usecase 層の責務です）。
type ListFeedParams struct {
	// Limit は、取得件数の上限です。
	Limit int32
	// AfterOrderedAt は、keyset 境界となる注文日時です。先頭ページでは nil です。
	AfterOrderedAt *time.Time
	// AfterID は、keyset 境界となる購入 ID です。先頭ページでは nil です。
	AfterID *uuid.UUID
}
