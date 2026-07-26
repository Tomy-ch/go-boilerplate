package purchase

import (
	"context"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// purchaseCursorKeyCount は、購入履歴一覧カーソルが保持するソートキーの個数（注文日時, ID）です。
const purchaseCursorKeyCount = 2

// PurchaseSummaryView は、購入履歴一覧の 1 件分のユースケース出力 DTO です。
// TotalAmount は USD セント単位の整数、ステータスは購入ステータスマスタで解決済みの ID と名称です。
type PurchaseSummaryView struct {
	Code        string
	TotalAmount int
	StatusID    uuid.UUID
	StatusName  string
	OrderedAt   time.Time
}

// PurchaseListView は、購入履歴一覧（cursor ページネーション）のユースケース出力 DTO です。
type PurchaseListView struct {
	// Items は、現在ページの購入履歴概要一覧です。
	Items []PurchaseSummaryView
	// NextCursor は、次ページ取得用の不透明カーソルです。最終ページの場合は nil です。
	NextCursor *string
}

// purchaseCursor は、購入履歴一覧（keyset ページネーション）の境界キーを表す usecase 層の値です。
// 直前ページ末尾行の注文日時と ID を保持し、次ページ取得時の keyset 比較の境界として用います。
// ドメイン層は不透明カーソルを持たず、この境界を primitive（注文日時, ID）で受け取ります。
type purchaseCursor struct {
	orderedAt time.Time
	id        uuid.UUID
}

// GetPurchases は、所有権の絞り込みを Repository に委ね、usecase 側では所有者を再判定しません。
// 対象が無い場合は Repository が空一覧を返すため、他ユーザーの購入が混ざる経路は存在しません。
func (u *usecase) GetPurchases(ctx context.Context, userID uuid.UUID, cursor *paging.Cursor) (*PurchaseListView, error) {
	if cursor == nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "cursor must not be nil")
	}

	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	after, err := decodePurchaseCursor(cursor)
	if err != nil {
		return nil, err
	}

	params := purchase.ListFeedParams{Limit: cursor.Limit32() + 1}
	if after != nil {
		orderedAt := after.orderedAt
		id := after.id
		params.AfterOrderedAt = &orderedAt
		params.AfterID = &id
	}

	feed, err := u.repo.FindFeedByUserID(ctx, userID, params)
	if err != nil {
		return nil, err
	}

	limit := cursor.Limit()
	hasNext := len(feed) > limit
	if hasNext {
		feed = feed[:limit]
	}

	items := make([]PurchaseSummaryView, len(feed))
	for i, f := range feed {
		items[i] = PurchaseSummaryView{
			Code:        f.Code,
			TotalAmount: f.TotalAmount,
			StatusID:    f.StatusID,
			StatusName:  f.StatusName,
			OrderedAt:   f.OrderedAt,
		}
	}

	var nextCursor *string
	if hasNext && len(feed) > 0 {
		encoded := encodePurchaseCursor(feed[len(feed)-1])
		nextCursor = &encoded
	}

	return &PurchaseListView{Items: items, NextCursor: nextCursor}, nil
}

// decodePurchaseCursor は、cursor の不透明キー列を keyset 境界（purchaseCursor）へ解釈します。
// 先頭ページ（カーソル無し）の場合は nil を返します。キーの個数・型が不正な場合は ErrInvalidArgument を返します。
func decodePurchaseCursor(cursor *paging.Cursor) (*purchaseCursor, error) {
	if !cursor.HasCursor() {
		return nil, nil //nolint:nilnil // 先頭ページは境界なし（nil）を正常値として返す
	}

	keys := cursor.Keys()
	if len(keys) != purchaseCursorKeyCount {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "invalid cursor: expected 2 keys")
	}

	orderedAt, err := time.Parse(time.RFC3339Nano, keys[0])
	if err != nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "invalid cursor: ordered_at is not RFC3339Nano")
	}

	id, err := uuid.Parse(keys[1])
	if err != nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "invalid cursor: id is not a valid UUID")
	}

	return &purchaseCursor{orderedAt: orderedAt, id: id}, nil
}

// encodePurchaseCursor は、現在ページ末尾のソートキー（注文日時, ID）から次ページ用の不透明カーソルを生成します。
func encodePurchaseCursor(last purchase.FeedItem) string {
	return paging.EncodeCursor(last.OrderedAt.Format(time.RFC3339Nano), last.ID.String())
}
