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

// DiscontinueProductParams は、商品を廃番にする要求の入力です。要求で条件を受ける理由は
// docs/spec/usecase/product.md の Workflow — DiscontinueProduct の notes を参照。
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
// 非公開化・進行中購入の判定・代替クーポンの一括発行を 1 トランザクションで行います。
// 分岐の判別根拠は ADR-0034 (commandservice-atomicity-criterion) の Worked instances、
// ロック順序と再実行時の規律は docs/spec/usecase/product.md の Workflow — DiscontinueProduct を参照。
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
