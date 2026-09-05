//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package query

import (
	"context"
	"time"

	"go-boilerplate/internal/usecase/tools/timewindow"
	"go-boilerplate/pkg/uuid"
)

// PurchaseFeedQueryService は、購入履歴一覧の集約跨ぎ read 投影を提供する QueryService です
// （配置根拠: docs/spec/usecase/purchase.md § GET 一覧）。
type PurchaseFeedQueryService interface {
	// FindFeedByUserID は、指定ユーザーの購入履歴を注文日時の降順（同時刻は ID 降順）の安定順で
	// keyset ページネーション取得します。ステータスは購入ステータスマスタ、明細の要約は商品との
	// 結合で解決します。所有権は本サービス側の絞り込みで担保します。
	// params.AfterOrderedAt / AfterID が nil の場合は先頭ページを返します。
	// params.Window が境界を持つ場合は、その半開区間に注文された購入だけを返します。
	// params.StatusCodes が空でない場合は、いずれかのステータスの購入だけを返します。
	FindFeedByUserID(ctx context.Context, userID uuid.UUID, params ListFeedParams) ([]PurchaseFeedReadModel, error)
	// FindFeedAll は、購入者を問わず購入履歴を FindFeedByUserID と同じ順序・同じ絞り込みで取得します。
	// 所有権で閉じないため、呼び出し側が可視範囲を認可したうえで用います。
	FindFeedAll(ctx context.Context, params ListFeedParams) ([]PurchaseFeedReadModel, error)
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
	// Window は、注文日時で絞り込む対象期間です。境界を持たない側には制限を設けません。
	Window timewindow.Window
	// StatusCodes は、購入ステータスの業務キーによる絞り込みです。いずれかに一致する購入を対象とし、
	// 空の場合は全ステータスを対象とします。既知でないコードは 0 件として扱います。
	StatusCodes []int16
	// ProductID は、指定商品を含む購入だけに絞る条件です。nil の場合は絞り込みません。
	// 廃番が進行中の購入を理由に拒まれたとき、どの購入が残っているのかを辿るために用います。
	ProductID *uuid.UUID
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
