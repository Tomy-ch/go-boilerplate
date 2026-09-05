package product

import (
	"context"
	"time"

	"go-boilerplate/internal/domain/coupon"
	"go-boilerplate/internal/domain/service/discontinuation"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	"go-boilerplate/internal/usecase/product/command"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/uuid"
)

// DiscontinueProductParams は、商品を廃番にする要求の入力です。
// 代替クーポンの条件を要求で受けるのは、何をどれだけ補償するかが業務の判断であり、
// この操作はその判断を実行するだけだからです。
type DiscontinueProductParams struct {
	// CouponDiscountRate は、代替クーポンの値引き率です。0 より大きく 1 以下である必要があります。
	CouponDiscountRate decimal.Decimal
	// CouponValidity は、代替クーポンの有効期間です。発行日時からこの長さだけ有効です。
	CouponValidity time.Duration
}

// DiscontinueProductView は、廃番の実行結果です。
type DiscontinueProductView struct {
	// DiscontinuedAt は、廃番が確定した日時です。
	DiscontinuedAt time.Time
	// AffectedCartCount は、対象商品の明細を持っていたカートの件数です。ゲストのカートも含みます。
	AffectedCartCount int64
	// AffectedUserCount は、クーポンの受給対象になった確定済みユーザーの数です。
	AffectedUserCount int64
	// IssuedCouponCount は、実際に発行したクーポンの枚数です。
	IssuedCouponCount int64
}

// DiscontinueImpactView は、廃番の影響の見積もりです。ロックを取らないため実行時の値と一致しません。
type DiscontinueImpactView struct {
	// AffectedCartCount は、対象商品の明細を持つカートの件数です。ゲストのカートも含みます。
	AffectedCartCount int64
	// AffectedUserCount は、クーポンの受給対象になる確定済みユーザーの数です。
	AffectedUserCount int64
	// InProgressPurchaseCount は、対象商品を含む進行中の購入の件数です。1 以上なら廃番は拒まれます。
	InProgressPurchaseCount int64
}

// DiscontinueProduct は、admin が商品を廃番にし、何が起きたかを件数で返します。
//
// 商品の非公開化と代替クーポンの一括発行を同一トランザクションで行います。クーポンだけが配られて
// 商品がまだ買える状態は observable になりません（ADR-0034 の branch 3）。
//
// 進行中の購入がある商品は拒否します（409）。この判定は商品行をロックしたうえで行うため、
// 購入作成が同じ商品行を id 順にロックする経路と直列化され、判定から commit まで覆りません
// （ADR-0034 の branch 2 / ADR-0036）。
//
// 既に廃番の商品への再実行は、新たな発行を伴わずに現在の状態と発行済みの枚数を返します。
// 明細を取り除かない設計のため、毎回発行すると同じ母集団へ重複して配ることになるためです。
func (u *usecase) DiscontinueProduct(
	ctx context.Context,
	authn *auth.Authn,
	id uuid.UUID,
	params DiscontinueProductParams,
) (DiscontinueProductView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if err := u.authorizer.Authorize(ctx, authn, authz.ActionProductDiscontinue, authz.NewResource("product", nil)); err != nil {
		return DiscontinueProductView{}, err
	}

	discount, err := coupon.NewRateDiscount(params.CouponDiscountRate)
	if err != nil {
		return DiscontinueProductView{}, err
	}

	now := u.clock.Now()

	var view DiscontinueProductView
	err = u.txm.Do(ctx, func(ctx context.Context) error {
		entity, lerr := u.repo.LockByID(ctx, id)
		if lerr != nil {
			return lerr
		}

		if entity.IsDiscontinued() {
			// 件数はいずれもこの実行が起こしたことを表すため、何も起こさない再実行では 0 のままにします。
			view = DiscontinueProductView{DiscontinuedAt: *entity.DiscontinuedAt()}

			return nil
		}

		statuses, perr := u.purchaseRepo.FindStatusesByProductID(ctx, id)
		if perr != nil {
			return perr
		}
		if derr := discontinuation.EnsureDiscontinuable(statuses); derr != nil {
			return derr
		}

		if derr := entity.Discontinue(now); derr != nil {
			return derr
		}
		if _, uerr := u.repo.Update(ctx, entity); uerr != nil {
			return uerr
		}

		// 適用範囲は廃番商品のカテゴリで固定です。廃番商品自身を範囲にすると買えない商品にしか
		// 使えないため、この journey が配るクーポンの範囲は 1 つに決まっています。
		scope, serr := coupon.NewCategoryScope(entity.Category().ID())
		if serr != nil {
			return serr
		}

		result, ierr := u.discontinueCmd.IssueDiscontinuationCoupons(ctx, command.IssueDiscontinuationCouponsParams{
			ProductID: id,
			Scope:     scope,
			Discount:  discount,
			ExpiresAt: now.Add(params.CouponValidity),
			IssuedAt:  now,
		})
		if ierr != nil {
			return ierr
		}

		view = DiscontinueProductView{
			DiscontinuedAt:    now,
			AffectedCartCount: result.AffectedCartCount,
			AffectedUserCount: result.AffectedUserCount,
			IssuedCouponCount: result.IssuedCouponCount,
		}

		return nil
	})
	if err != nil {
		return DiscontinueProductView{}, err
	}

	return view, nil
}

// GetDiscontinueImpact は、admin が商品を廃番にした場合の影響を実行前に件数で取得します。
// 行をロックしないため、返した値は実行時の件数と一致する保証を持ちません。
func (u *usecase) GetDiscontinueImpact(
	ctx context.Context,
	authn *auth.Authn,
	id uuid.UUID,
) (DiscontinueImpactView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if err := u.authorizer.Authorize(ctx, authn, authz.ActionProductDiscontinue, authz.NewResource("product", nil)); err != nil {
		return DiscontinueImpactView{}, err
	}

	// 商品の存在は見積もりの前に確かめます。存在しない商品の影響は 0 件ではなく引けないためです。
	if _, err := u.repo.FindByID(ctx, id); err != nil {
		return DiscontinueImpactView{}, err
	}

	impact, err := u.discontinueImpactQuery.EstimateDiscontinueImpact(ctx, id)
	if err != nil {
		return DiscontinueImpactView{}, err
	}

	return DiscontinueImpactView{
		AffectedCartCount:       impact.AffectedCartCount,
		AffectedUserCount:       impact.AffectedUserCount,
		InProgressPurchaseCount: impact.InProgressPurchaseCount,
	}, nil
}
