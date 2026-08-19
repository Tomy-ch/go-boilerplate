//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package query

import (
	"context"
	"time"

	"go-boilerplate/pkg/uuid"
)

// PurchaseFeedQueryService は、購入履歴一覧の集約跨ぎ read 投影を提供する QueryService です。
// 一覧は購入ステータスマスタに加えて別集約である商品（products）を結合して明細の要約を解決するため、
// 単一集約の Repository read ではなく読み取り側に置きます（ADR-0031 (lightweight-cqrs)）。
type PurchaseFeedQueryService interface {
	// FindFeedByUserID は、指定ユーザーの購入履歴を注文日時の降順（同時刻は ID 降順）の安定順で
	// keyset ページネーション取得します。ステータスは購入ステータスマスタ、明細の要約は商品との
	// 結合で解決します。所有権は本サービス側の絞り込みで担保します。
	// params.AfterOrderedAt / AfterID が nil の場合は先頭ページを返します。
	// params.OrderedAfter / OrderedBefore が揃っている場合は、その半開区間に注文された購入だけを返します。
	FindFeedByUserID(ctx context.Context, userID uuid.UUID, params ListFeedParams) ([]PurchaseFeedReadModel, error)
}

// ListFeedParams は、購入履歴フィード（keyset ページネーション）の取得条件です。
// 各フィールドの意味は FindFeedByUserID を参照（不透明カーソルの符号化・復号は呼び出し側の責務です）。
type ListFeedParams struct {
	// Limit は、取得件数の上限です。
	Limit int32
	// AfterOrderedAt は、keyset 境界となる注文日時です。先頭ページでは nil です。
	AfterOrderedAt *time.Time
	// AfterID は、keyset 境界となる購入 ID です。先頭ページでは nil です。
	AfterID *uuid.UUID
	// OrderedAfter / OrderedBefore は、注文日時で絞り込む半開区間 [OrderedAfter, OrderedBefore) です。
	// 期間で絞り込まない場合はいずれも nil で、片方だけを指定した場合も絞り込みません。
	OrderedAfter  *time.Time
	OrderedBefore *time.Time
}

// PurchaseFeedReadModel は、購入履歴一覧の 1 件分の読み取りモデルです。
// 一覧は概要のみを返し明細は含みませんが、行を見分けるための要約 2 項目を持ちます。
type PurchaseFeedReadModel struct {
	// Code は、購入コード（利用者へ注文番号として見せる一意の識別子）です。
	Code string
	// TotalAmount は、合計金額（小計 + 税額 + 送料）です。USD セント単位の整数です。
	TotalAmount int
	// StatusID は、購入ステータス ID（購入ステータスマスタで解決済み）です。
	StatusID uuid.UUID
	// StatusCode は、購入ステータスの業務キー（Status.Code）です。到達順序を意味しません。
	StatusCode int
	// StatusName は、購入ステータスの名称（購入ステータスマスタで解決済み）です。
	StatusName string
	// FirstItemName は、明細の先頭 1 件（明細 ID 昇順）の商品名です。先頭の選び方に業務的な意味はありません。
	FirstItemName string
	// ItemCount は、明細の行数です（先頭商品を含む）。数量の合計ではありません。
	ItemCount int
	// OrderedAt は、注文日時です。keyset の主ソートキーです。
	OrderedAt time.Time
	// ID は、購入 ID です。keyset のタイブレークキーであり、応答へは出しません。
	ID uuid.UUID
}
