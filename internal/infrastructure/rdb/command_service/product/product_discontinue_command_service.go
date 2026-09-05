// Package product は、廃番のコマンドサービス（command.CommandService）の RDB 実装を提供します。
package product

import (
	"context"

	"go-boilerplate/internal/domain/coupon"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/product/command"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/uuid"
)

type commandService struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、廃番のコマンドサービスの RDB 実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) command.CommandService {
	return &commandService{
		db:     db,
		tracer: tf.Infra(),
	}
}

// IssueDiscontinuationCoupons は、対象商品の明細を持つカートの所有者へクーポンを一括発行します。
//
// 受給者の取得と挿入の 2 文で構成します。id をドメイン層で採番する（ADR-0037）ため件数を先に知る
// 必要があり、1 文にはできません。ただし往復は発行枚数に依存せず 2 回で固定なので、集合演算として
// 満たすべき性質（N 非依存）は保たれます。
//
// 受給者が 0 人の場合は挿入を行いません。空配列を unnest しても 0 行ですが、無駄な往復を避けます。
func (s *commandService) IssueDiscontinuationCoupons(
	ctx context.Context,
	params command.IssueDiscontinuationCouponsParams,
) (command.IssueDiscontinuationCouponsResult, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))

	affectedCarts, err := db.CountDiscontinueAffectedCarts(ctx, params.ProductID)
	if err != nil {
		return command.IssueDiscontinuationCouponsResult{}, pgerror.NormalizeError(err)
	}

	recipients, err := db.SelectDiscontinueCouponRecipients(ctx, params.ProductID)
	if err != nil {
		return command.IssueDiscontinuationCouponsResult{}, pgerror.NormalizeError(err)
	}
	if len(recipients) == 0 {
		return command.IssueDiscontinuationCouponsResult{AffectedCartCount: affectedCarts}, nil
	}

	ids := make([]uuid.UUID, len(recipients))
	for i := range recipients {
		id, ierr := uuid.New()
		if ierr != nil {
			return command.IssueDiscontinuationCouponsResult{}, ierr
		}
		ids[i] = id
	}

	discountKind, err := safecast.IntToInt16(params.Discount.Kind().Code())
	if err != nil {
		return command.IssueDiscontinuationCouponsResult{}, err
	}
	// 適用範囲は廃番商品のカテゴリで固定です。廃番商品自身を範囲にすると買えない商品にしか
	// 使えないため、この journey が配るクーポンの範囲は 1 つに決まっています。
	scopeKind, err := safecast.IntToInt16(coupon.ScopeKindCategory.Code())
	if err != nil {
		return command.IssueDiscontinuationCouponsResult{}, err
	}

	issued, err := db.InsertDiscontinueCoupons(ctx, &gen.InsertDiscontinueCouponsParams{
		Ids:           ids,
		UserIds:       recipients,
		DiscountKind:  discountKind,
		DiscountValue: params.Discount.Value(),
		ScopeKind:     scopeKind,
		ScopeTargetID: &params.CategoryID,
		ExpiresAt:     params.ExpiresAt,
		IssuedAt:      params.IssuedAt,
	})
	if err != nil {
		return command.IssueDiscontinuationCouponsResult{}, pgerror.NormalizeError(err)
	}

	return command.IssueDiscontinuationCouponsResult{
		AffectedCartCount: affectedCarts,
		AffectedUserCount: int64(len(recipients)),
		IssuedCouponCount: issued,
	}, nil
}
