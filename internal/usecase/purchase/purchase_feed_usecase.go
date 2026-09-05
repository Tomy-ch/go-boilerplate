package purchase

import (
	"context"
	"fmt"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	"go-boilerplate/internal/usecase/purchase/query"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/internal/usecase/tools/timewindow"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// purchaseCursorKeyCount は、購入履歴一覧カーソルが保持するソートキーの個数（注文日時, ID）です。
const purchaseCursorKeyCount = 2

// PurchaseSummaryView は、購入履歴一覧の 1 件分のユースケース出力 DTO です。TotalAmount は USD セント
// 単位の整数、ステータスは購入ステータスマスタで解決済みの ID・業務キー・名称、FirstItemName /
// ItemCount は行を見分けるための明細の要約です。
type PurchaseSummaryView struct {
	Code          string
	TotalAmount   int
	StatusID      uuid.UUID
	StatusCode    int
	StatusName    string
	FirstItemName string
	ItemCount     int
	OrderedAt     time.Time
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

// ListPurchasesParams は、購入履歴一覧取得の入力パラメータです。
type ListPurchasesParams struct {
	// Cursor は、keyset ページネーションの位置と件数です。nil は不正な入力として扱います。
	Cursor *paging.Cursor
	// Window は、注文日時で絞り込む対象期間です。境界を持たない側には条件を付けません。
	Window timewindow.Window
	// StatusCodes は、購入ステータスの業務キーによる絞り込みです。空の場合は全ステータスが対象です。
	StatusCodes []int16
	// ProductID は、指定商品を含む購入だけに絞る条件です。nil の場合は絞り込みません。
	ProductID *uuid.UUID
	// IncludeOtherUsers は、他ユーザーの購入も母集団に含める場合に true です。指定は admin のみが通ります。
	IncludeOtherUsers bool
}

// GetPurchases は、既定では所有権の絞り込みを QueryService に委ね、usecase 側では所有者を再判定しません。
// 対象が無い場合は QueryService が空一覧を返すため、他ユーザーの購入が混ざる経路は存在しません
// （母集団の切り替え条件は Usecase.GetPurchases のインターフェース doc を参照）。
// 母集団が変わっても並び順の軸は変わらないため、カーソルの解釈は共通です。
// 注文日時の対象期間は params.Window をそのまま QueryService へ渡し、境界を持たない側には条件を付けません。
func (u *usecase) GetPurchases(
	ctx context.Context, authn *auth.Authn, params ListPurchasesParams,
) (*PurchaseListView, error) {
	if params.Cursor == nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "cursor must not be nil")
	}
	if authn == nil {
		return nil, apperror.ErrUnauthenticated
	}

	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	cursor := params.Cursor
	after, err := decodePurchaseCursor(cursor)
	if err != nil {
		return nil, err
	}

	feedParams := query.ListFeedParams{
		Limit:       cursor.Limit32() + 1,
		Window:      params.Window,
		StatusCodes: params.StatusCodes,
		ProductID:   params.ProductID,
	}
	if after != nil {
		feedParams.AfterOrderedAt = &after.orderedAt
		feedParams.AfterID = &after.id
	}

	feed, err := u.findFeedPage(ctx, authn, params.IncludeOtherUsers, feedParams)
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
			Code:          f.Code,
			TotalAmount:   f.TotalAmount,
			StatusID:      f.StatusID,
			StatusCode:    f.StatusCode,
			StatusName:    f.StatusName,
			FirstItemName: f.FirstItemName,
			ItemCount:     f.ItemCount,
			OrderedAt:     f.OrderedAt,
		}
	}

	var nextCursor *string
	if hasNext && len(feed) > 0 {
		// len 判定は防御的な安全弁。hasNext は len > limit なので limit >= 1 の下では冗長だが、
		// limit の下限保証は paging.NewCursor 依存であり、ゼロ値 Cursor 混入時の feed[-1] panic を防ぐ。
		encoded := encodePurchaseCursor(feed[len(feed)-1])
		nextCursor = &encoded
	}

	return &PurchaseListView{Items: items, NextCursor: nextCursor}, nil
}

// findFeedPage は、可視範囲に応じた読み取りでフィードの 1 ページを取得します。既定（自分の購入のみ）は
// 認証済みであること以上を要求せず、他ユーザーを含める指定のときだけ admin の能力を要求します
// （設計意図: docs/spec/usecase/purchase.md § GET 一覧）。
func (u *usecase) findFeedPage(
	ctx context.Context, authn *auth.Authn, includeOtherUsers bool, params query.ListFeedParams,
) ([]query.PurchaseFeedReadModel, error) {
	if !includeOtherUsers {
		userID, err := authn.UserID()
		if err != nil {
			return nil, xerrors.Wrap(err, "failed to get user ID from authn")
		}
		return u.feedQS.FindFeedByUserID(ctx, userID, params)
	}

	if err := u.authorizer.Authorize(
		ctx, authn, authz.ActionPurchaseReadAll, authz.NewResource("purchase", nil),
	); err != nil {
		return nil, err
	}

	return u.feedQS.FindFeedAll(ctx, params)
}

// decodePurchaseCursor は、cursor の不透明キー列を keyset 境界（purchaseCursor）へ解釈します。
// 先頭ページ（カーソル無し）の場合は nil を返します。キーの個数・型が不正な場合は ErrInvalidArgument を返します。
func decodePurchaseCursor(cursor *paging.Cursor) (*purchaseCursor, error) {
	if !cursor.HasCursor() {
		return nil, nil //nolint:nilnil // 先頭ページは境界なし（nil）を正常値として返す
	}

	keys := cursor.Keys()
	if len(keys) != purchaseCursorKeyCount {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, fmt.Sprintf("invalid cursor: expected %d keys", purchaseCursorKeyCount))
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
func encodePurchaseCursor(last query.PurchaseFeedReadModel) string {
	return paging.EncodeCursor(last.OrderedAt.Format(time.RFC3339Nano), last.ID.String())
}
