package product

import (
	"testing"
	"time"

	domaincoupon "go-boilerplate/internal/domain/coupon"
	mock_coupon "go-boilerplate/internal/domain/coupon/mock"
	domainproduct "go-boilerplate/internal/domain/product"
	mock_product "go-boilerplate/internal/domain/product/mock"
	domainpurchase "go-boilerplate/internal/domain/purchase"
	mock_purchase "go-boilerplate/internal/domain/purchase/mock"
	"go-boilerplate/internal/domain/service/discontinuation"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	mock_authz "go-boilerplate/internal/usecase/boundary/authz/mock"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"
	"go-boilerplate/internal/usecase/product/command"
	mock_command "go-boilerplate/internal/usecase/product/command/mock"
	"go-boilerplate/internal/usecase/product/query"
	mock_query "go-boilerplate/internal/usecase/product/query/mock"
	"go-boilerplate/pkg/decimal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// discontinueTestDeps は、廃番のテストで注入する依存モック一式です。
type discontinueTestDeps struct {
	txm          *mock_tx.MockManager
	repo         *mock_product.MockRepository
	purchaseRepo *mock_purchase.MockRepository
	couponRepo   *mock_coupon.MockRepository
	cmd          *mock_command.MockCommandService
	impactQuery  *mock_query.MockDiscontinueImpactQueryService
	authorizer   *mock_authz.MockAuthorizer
	clock        *mock_clock.MockClock
}

func newDiscontinueTestUsecase(t *testing.T) (*usecase, *discontinueTestDeps) {
	t.Helper()

	ctrl := gomock.NewController(t)
	deps := &discontinueTestDeps{
		txm:          mock_tx.NewMockManager(ctrl),
		repo:         mock_product.NewMockRepository(ctrl),
		purchaseRepo: mock_purchase.NewMockRepository(ctrl),
		couponRepo:   mock_coupon.NewMockRepository(ctrl),
		cmd:          mock_command.NewMockCommandService(ctrl),
		impactQuery:  mock_query.NewMockDiscontinueImpactQueryService(ctrl),
		authorizer:   mock_authz.NewMockAuthorizer(ctrl),
		clock:        mock_clock.NewMockClock(ctrl),
	}
	u := &usecase{
		tracer:                 observability.NewMockUsecaseLayerTracer(t),
		txm:                    deps.txm,
		repo:                   deps.repo,
		authorizer:             deps.authorizer,
		clock:                  deps.clock,
		purchaseRepo:           deps.purchaseRepo,
		couponRepo:             deps.couponRepo,
		discontinueCmd:         deps.cmd,
		discontinueImpactQuery: deps.impactQuery,
	}

	return u, deps
}

func newDiscontinueRate(t *testing.T) decimal.Decimal {
	t.Helper()
	d, err := decimal.Parse("0.10")
	require.NoError(t, err)

	return d
}

func newTerminalStatus(t *testing.T) domainpurchase.Status {
	t.Helper()
	s, err := domainpurchase.NewStatus(domainpurchase.StatusCompleted.Code())
	require.NoError(t, err)

	return s
}

func newInProgressStatus(t *testing.T) domainpurchase.Status {
	t.Helper()
	s, err := domainpurchase.NewStatus(domainpurchase.StatusPaid.Code())
	require.NoError(t, err)

	return s
}

func Test_usecase_DiscontinueProduct(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.September, 5, 0, 0, 0, 0, time.UTC)
	params := DiscontinueProductParams{CouponDiscountRate: newDiscontinueRate(t), CouponValidity: 30 * 24 * time.Hour}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("進行中の購入が無い場合、非公開化とクーポンの一括発行を行い件数を返す", func(t *testing.T) {
			t.Parallel()

			entity := newUpdateTarget(t, 1)
			u, deps := newDiscontinueTestUsecase(t)

			deps.authorizer.EXPECT().
				Authorize(gomock.Any(), gomock.Any(), authz.ActionProductDiscontinue, gomock.Any()).
				Return(nil)
			deps.clock.EXPECT().Now().Return(now)
			deps.txm.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(runInTx)
			deps.repo.EXPECT().LockByID(gomock.Any(), entity.ID()).Return(entity, nil)
			deps.purchaseRepo.EXPECT().
				FindStatusesByProductID(gomock.Any(), entity.ID()).
				Return([]domainpurchase.Status{newTerminalStatus(t)}, nil)
			deps.repo.EXPECT().Update(gomock.Any(), entity).Return(2, nil)
			deps.cmd.EXPECT().
				IssueDiscontinuationCoupons(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ any, p command.IssueDiscontinuationCouponsParams) (command.IssueDiscontinuationCouponsResult, error) {
					assert.Equal(t, entity.ID(), p.ProductID)
					assert.Equal(t, entity.Category().ID(), p.CategoryID)
					assert.Equal(t, domaincoupon.DiscountKindRate, p.Discount.Kind())
					assert.Equal(t, now.Add(30*24*time.Hour), p.ExpiresAt)
					assert.Equal(t, now, p.IssuedAt)

					return command.IssueDiscontinuationCouponsResult{
						AffectedCartCount: 12, AffectedUserCount: 9, IssuedCouponCount: 9,
					}, nil
				})

			actual, err := u.DiscontinueProduct(t.Context(), &auth.Authn{}, entity.ID(), params)

			require.NoError(t, err)
			assert.Equal(t, now, actual.DiscontinuedAt)
			assert.Equal(t, int64(12), actual.AffectedCartCount)
			assert.Equal(t, int64(9), actual.AffectedUserCount)
			assert.Equal(t, int64(9), actual.IssuedCouponCount)
			assert.True(t, entity.IsDiscontinued())
			assert.False(t, entity.IsPublished())
		})

		t.Run("既に廃番の場合、発行を行わず発行済みの枚数を返す", func(t *testing.T) {
			t.Parallel()

			entity := newUpdateTarget(t, 1)
			require.NoError(t, entity.Discontinue(now))
			u, deps := newDiscontinueTestUsecase(t)

			deps.authorizer.EXPECT().
				Authorize(gomock.Any(), gomock.Any(), authz.ActionProductDiscontinue, gomock.Any()).
				Return(nil)
			deps.clock.EXPECT().Now().Return(now)
			deps.txm.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(runInTx)
			deps.repo.EXPECT().LockByID(gomock.Any(), entity.ID()).Return(entity, nil)
			deps.couponRepo.EXPECT().CountByScopeTargetProductID(gomock.Any(), entity.ID()).Return(9, nil)
			// 発行も更新も行わないことを、EXPECT を置かないことで表す。

			actual, err := u.DiscontinueProduct(t.Context(), &auth.Authn{}, entity.ID(), params)

			require.NoError(t, err)
			assert.Equal(t, now, actual.DiscontinuedAt)
			assert.Equal(t, int64(9), actual.IssuedCouponCount)
			assert.Zero(t, actual.AffectedUserCount)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("進行中の購入がある場合、発行を行わずErrInProgressPurchaseExistsを返す", func(t *testing.T) {
			t.Parallel()

			entity := newUpdateTarget(t, 1)
			u, deps := newDiscontinueTestUsecase(t)

			deps.authorizer.EXPECT().
				Authorize(gomock.Any(), gomock.Any(), authz.ActionProductDiscontinue, gomock.Any()).
				Return(nil)
			deps.clock.EXPECT().Now().Return(now)
			deps.txm.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(runInTx)
			deps.repo.EXPECT().LockByID(gomock.Any(), entity.ID()).Return(entity, nil)
			deps.purchaseRepo.EXPECT().
				FindStatusesByProductID(gomock.Any(), entity.ID()).
				Return([]domainpurchase.Status{newInProgressStatus(t)}, nil)

			_, err := u.DiscontinueProduct(t.Context(), &auth.Authn{}, entity.ID(), params)

			require.ErrorIs(t, err, discontinuation.ErrInProgressPurchaseExists)
			assert.False(t, entity.IsDiscontinued())
		})

		t.Run("値引き率が範囲外の場合、認可の後に検証で弾き商品を引かない", func(t *testing.T) {
			t.Parallel()

			u, deps := newDiscontinueTestUsecase(t)
			deps.authorizer.EXPECT().
				Authorize(gomock.Any(), gomock.Any(), authz.ActionProductDiscontinue, gomock.Any()).
				Return(nil)

			invalid := DiscontinueProductParams{CouponDiscountRate: decimal.FromInt(0), CouponValidity: time.Hour}

			_, err := u.DiscontinueProduct(t.Context(), &auth.Authn{}, newUpdateTarget(t, 1).ID(), invalid)

			require.ErrorIs(t, err, domaincoupon.ErrInvalidDiscountValue)
		})

		t.Run("認可に失敗した場合、時刻も取得せずエラーを返す", func(t *testing.T) {
			t.Parallel()

			u, deps := newDiscontinueTestUsecase(t)
			deps.authorizer.EXPECT().
				Authorize(gomock.Any(), gomock.Any(), authz.ActionProductDiscontinue, gomock.Any()).
				Return(authz.ErrForbidden)

			_, err := u.DiscontinueProduct(t.Context(), &auth.Authn{}, newUpdateTarget(t, 1).ID(), params)

			require.ErrorIs(t, err, authz.ErrForbidden)
		})
	})
}

func Test_usecase_GetDiscontinueImpact(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("見積もりの 3 つの件数をそのまま返す", func(t *testing.T) {
			t.Parallel()

			entity := newUpdateTarget(t, 1)
			u, deps := newDiscontinueTestUsecase(t)

			deps.authorizer.EXPECT().
				Authorize(gomock.Any(), gomock.Any(), authz.ActionProductDiscontinue, gomock.Any()).
				Return(nil)
			deps.repo.EXPECT().FindByID(gomock.Any(), entity.ID()).Return(entity, nil)
			deps.impactQuery.EXPECT().
				EstimateDiscontinueImpact(gomock.Any(), entity.ID()).
				Return(query.DiscontinueImpactReadModel{
					AffectedCartCount: 12, AffectedUserCount: 9, InProgressPurchaseCount: 2,
				}, nil)

			actual, err := u.GetDiscontinueImpact(t.Context(), &auth.Authn{}, entity.ID())

			require.NoError(t, err)
			assert.Equal(t, int64(12), actual.AffectedCartCount)
			assert.Equal(t, int64(9), actual.AffectedUserCount)
			assert.Equal(t, int64(2), actual.InProgressPurchaseCount)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("商品が存在しない場合、見積もりを引かずエラーを返す", func(t *testing.T) {
			t.Parallel()

			entity := newUpdateTarget(t, 1)
			u, deps := newDiscontinueTestUsecase(t)

			deps.authorizer.EXPECT().
				Authorize(gomock.Any(), gomock.Any(), authz.ActionProductDiscontinue, gomock.Any()).
				Return(nil)
			deps.repo.EXPECT().FindByID(gomock.Any(), entity.ID()).Return(nil, domainproduct.ErrVersionConflict)

			_, err := u.GetDiscontinueImpact(t.Context(), &auth.Authn{}, entity.ID())

			require.ErrorIs(t, err, domainproduct.ErrVersionConflict)
		})
	})
}
