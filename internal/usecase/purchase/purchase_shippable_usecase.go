package purchase

import (
	"context"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/domain/service/dispatch"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

const (
	// shippableDefaultLimit は、limit 未指定時に用いる既定の取得件数です。
	shippableDefaultLimit = 20
	// shippableMaxLimit は、許容する最大の取得件数です。
	shippableMaxLimit = 100
)

// shippableLimitPolicy は、発送待ち購入一覧の取得件数規約です。OpenAPI の limit（既定 20 / 1〜100）と対応します。
var (
	shippableLimitPolicy = paging.LimitPolicy{Default: shippableDefaultLimit, Max: shippableMaxLimit}

	// errNotShippableInShippableRead は、発送可能として取得した読み取りに該当しない購入が混じっていた場合の
	// エラーです。絞り込みを実行する SQL と、発送可能を定義する Purchase.IsShippable が食い違ったことを
	// 意味します。
	errNotShippableInShippableRead = xerrors.Wrap(apperror.ErrInternal, "purchase not shippable in shippable read")
)

// ListShippablePurchasesParams は、発送待ち購入一覧取得の入力パラメータです。
type ListShippablePurchasesParams struct {
	// Limit は、読み出す購入の件数です。1 未満は既定値 20 を適用し、100 を超える値は 100 にクランプします。
	Limit int
}

// ShippablePurchaseView は、まとめ発送の組に含まれる購入 1 件のユースケース出力 DTO です。
// 金額は USD セント単位の整数です。
type ShippablePurchaseView struct {
	ID          uuid.UUID
	Code        string
	TotalAmount int
	OrderedAt   time.Time
}

// DispatchGroupView は、まとめて発送してよい購入の組のユースケース出力 DTO です。
type DispatchGroupView struct {
	// UserID は、組に含まれる購入の購入者 ID です。組の中では全件が同一です。
	UserID uuid.UUID
	// Purchases は、組に含まれる購入です。注文日時の古い順（同時刻は購入 ID の昇順）で並びます。
	Purchases []ShippablePurchaseView
}

// PurchaseShippableListView は、発送待ち購入一覧の取得結果を表します。
type PurchaseShippableListView struct {
	// Groups は、まとめ発送の組の一覧です。組同士はその組の最も古い購入の順に並びます。
	Groups []DispatchGroupView
}

// ListShippablePurchases は、admin 認可のうえ、発送可能な購入をまとめて発送してよい組に分けて返します。
func (u *usecase) ListShippablePurchases(
	ctx context.Context, authn *auth.Authn, params ListShippablePurchasesParams,
) (PurchaseShippableListView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if authn == nil {
		return PurchaseShippableListView{}, apperror.ErrUnauthenticated
	}
	if err := u.authorizer.Authorize(
		ctx, authn, authz.ActionPurchaseListShippable, authz.NewResource("purchase", nil),
	); err != nil {
		return PurchaseShippableListView{}, err
	}

	limit := paging.NewLimit(ptr.To(params.Limit), shippableLimitPolicy)
	purchases, err := u.repo.FindShippable(ctx, limit.Value32())
	if err != nil {
		return PurchaseShippableListView{}, err
	}
	// 絞り込みを実行するのは SQL だが、発送可能を定義するのは Purchase.IsShippable。両者の乖離を表に出す。
	for _, p := range purchases {
		if !p.IsShippable() {
			return PurchaseShippableListView{}, xerrors.Wrap(errNotShippableInShippableRead, p.ID().String())
		}
	}

	groups := dispatch.GroupForDispatch(purchases)

	views := make([]DispatchGroupView, len(groups))
	for i, group := range groups {
		views[i] = toDispatchGroupView(group)
	}

	return PurchaseShippableListView{Groups: views}, nil
}

// toDispatchGroupView は、まとめ発送の組を出力 DTO へ写像します。
// 組は同一購入者で分けられているため、購入者 ID は先頭の購入から取ります。
func toDispatchGroupView(group purchase.Purchases) DispatchGroupView {
	items := make([]ShippablePurchaseView, len(group))
	for i, p := range group {
		items[i] = ShippablePurchaseView{
			ID:          p.ID(),
			Code:        p.Code(),
			TotalAmount: p.TotalAmount(),
			OrderedAt:   p.OrderedAt(),
		}
	}
	return DispatchGroupView{UserID: group[0].UserID(), Purchases: items}
}
