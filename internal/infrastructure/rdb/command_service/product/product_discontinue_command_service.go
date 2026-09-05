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
// 挿入する行は必ず coupon.New を通して組み立てます。集合演算で書くことと、書く 1 行 1 行が集約の
// 不変条件を満たすことは両立します。列へ展開するのは検証を終えた集約からで、素の値からではありません。
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

	coupons := make(coupon.Coupons, len(recipients))
	ids := make([]uuid.UUID, len(recipients))
	for i, recipient := range recipients {
		id, ierr := uuid.New()
		if ierr != nil {
			return command.IssueDiscontinuationCouponsResult{}, ierr
		}

		issuedCoupon, cerr := coupon.New(id, coupon.Attributes{
			UserID:    recipient,
			Discount:  params.Discount,
			Scope:     params.Scope,
			ExpiresAt: params.ExpiresAt,
			IssuedAt:  params.IssuedAt,
		})
		if cerr != nil {
			return command.IssueDiscontinuationCouponsResult{}, cerr
		}

		coupons[i] = issuedCoupon
		ids[i] = issuedCoupon.ID()
	}

	// 全員に同じ条件で配るため、共有の列は先頭の集約から取ります。素の params ではなく検証を
	// 通った集約を出所にすることで、書き込む値と検証した値が同じであることが保証されます。
	template := coupons[0]

	discountKind, err := safecast.IntToInt16(template.Discount().Kind().Code())
	if err != nil {
		return command.IssueDiscontinuationCouponsResult{}, err
	}
	scopeKind, err := safecast.IntToInt16(template.Scope().Kind().Code())
	if err != nil {
		return command.IssueDiscontinuationCouponsResult{}, err
	}

	issued, err := db.InsertDiscontinueCoupons(ctx, &gen.InsertDiscontinueCouponsParams{
		Ids:           ids,
		UserIds:       recipients,
		DiscountKind:  discountKind,
		DiscountValue: template.Discount().Value(),
		ScopeKind:     scopeKind,
		ScopeTargetID: template.Scope().TargetID(),
		ExpiresAt:     template.ExpiresAt(),
		IssuedAt:      template.IssuedAt(),
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
