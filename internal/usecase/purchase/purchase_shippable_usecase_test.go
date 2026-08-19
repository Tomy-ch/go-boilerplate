package purchase

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/lexicon/money"
	domainpurchase "go-boilerplate/internal/domain/purchase"
	mock_purchase "go-boilerplate/internal/domain/purchase/mock"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	mock_authz "go-boilerplate/internal/usecase/boundary/authz/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// shippableBaseTime は、テスト用の購入の注文日時の基準です。
var shippableBaseTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

// shippableTestDeps は、ListShippablePurchases のテストで注入する依存モック一式です。
type shippableTestDeps struct {
	repo       *mock_purchase.MockRepository
	authorizer *mock_authz.MockAuthorizer
}

// newShippableTestUsecase は、モック依存のみで構成した usecase とそのモック一式を返します。
func newShippableTestUsecase(t *testing.T) (*usecase, *shippableTestDeps) {
	t.Helper()

	ctrl := gomock.NewController(t)
	deps := &shippableTestDeps{
		repo:       mock_purchase.NewMockRepository(ctrl),
		authorizer: mock_authz.NewMockAuthorizer(ctrl),
	}
	u := &usecase{
		tracer:     observability.NewMockUsecaseLayerTracer(t),
		repo:       deps.repo,
		authorizer: deps.authorizer,
	}

	return u, deps
}

// newShippablePurchase は、発送可能（支払い済み）な購入を再構築します。
func newShippablePurchase(
	t *testing.T, salt string, userID uuid.UUID, orderedAt time.Time,
) *domainpurchase.Purchase {
	t.Helper()
	return newPurchaseWithStatus(t, salt, userID, orderedAt, domainpurchase.StatusPaid)
}

// newPurchaseWithStatus は、指定ステータスの購入を再構築します。発送可能でない購入を作るのに用います。
func newPurchaseWithStatus(
	t *testing.T, salt string, userID uuid.UUID, orderedAt time.Time, status domainpurchase.Status,
) *domainpurchase.Purchase {
	t.Helper()

	unitPrice, err := money.NewPrice(decimaltestkit.MustParse(t, "500"))
	require.NoError(t, err)

	paidAt := orderedAt.Add(time.Minute)
	attrs := domainpurchase.Attributes{
		Code:       "code-" + salt,
		UserID:     userID,
		StatusID:   uuidtestkit.NewTestFromSalt(t, "shippable_status"),
		StatusCode: status.Code(),
		Details: []domainpurchase.PurchaseDetail{
			domainpurchase.NewPurchaseDetail(
				uuidtestkit.NewTestFromSalt(t, salt+"_detail"),
				domainpurchase.PurchaseDetailAttributes{
					ProductID: uuidtestkit.NewTestFromSalt(t, "shippable_product"),
					Quantity:  1,
					UnitPrice: unitPrice,
				},
			),
		},
		SubtotalAmount: 50000,
		TaxAmount:      5000,
		ShippingFee:    500,
		TotalAmount:    55500,
		OrderedAt:      orderedAt,
		PaidAt:         &paidAt,
	}
	if status == domainpurchase.StatusShipped {
		shippedAt := orderedAt.Add(time.Hour)
		attrs.ShippedAt = &shippedAt
	}

	p, err := domainpurchase.Reconstruct(uuidtestkit.NewTestFromSalt(t, salt), attrs)
	require.NoError(t, err)
	return p
}

func Test_toDispatchGroupView(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("組の購入者と購入一覧を順序どおりDTOへ写す", func(t *testing.T) {
			t.Parallel()

			alice := uuidtestkit.NewTestFromSalt(t, "view_alice")
			first := newShippablePurchase(t, "view_p1", alice, shippableBaseTime)
			second := newShippablePurchase(t, "view_p2", alice, shippableBaseTime.Add(time.Hour))

			actual := toDispatchGroupView(domainpurchase.Purchases{first, second})

			assert.Equal(t, alice, actual.UserID)
			require.Len(t, actual.Purchases, 2)
			assert.Equal(t, first.Code(), actual.Purchases[0].Code)
			assert.Equal(t, first.TotalAmount(), actual.Purchases[0].TotalAmount)
			assert.Equal(t, first.OrderedAt(), actual.Purchases[0].OrderedAt)
			assert.Equal(t, second.Code(), actual.Purchases[1].Code)
		})
	})
}

func Test_usecase_ListShippablePurchases(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購入者ごとの組へ分けてDTOへ写す", func(t *testing.T) {
			t.Parallel()

			u, deps := newShippableTestUsecase(t)
			alice := uuidtestkit.NewTestFromSalt(t, "uc_alice")
			bob := uuidtestkit.NewTestFromSalt(t, "uc_bob")
			a1 := newShippablePurchase(t, "uc_a1", alice, shippableBaseTime)
			b1 := newShippablePurchase(t, "uc_b1", bob, shippableBaseTime.Add(time.Hour))
			a2 := newShippablePurchase(t, "uc_a2", alice, shippableBaseTime.Add(2*time.Hour))

			deps.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			deps.repo.EXPECT().FindShippable(gomock.Any(), gomock.Any()).
				Return(domainpurchase.Purchases{a1, b1, a2}, nil)

			actual, err := u.ListShippablePurchases(
				context.Background(), &auth.Authn{}, ListShippablePurchasesParams{Limit: 20},
			)
			require.NoError(t, err)

			require.Len(t, actual.Groups, 2)
			assert.Equal(t, alice, actual.Groups[0].UserID)
			require.Len(t, actual.Groups[0].Purchases, 2)
			assert.Equal(t, a1.Code(), actual.Groups[0].Purchases[0].Code)
			assert.Equal(t, a2.Code(), actual.Groups[0].Purchases[1].Code)
			assert.Equal(t, bob, actual.Groups[1].UserID)
			require.Len(t, actual.Groups[1].Purchases, 1)
		})

		t.Run("発送待ち一覧の参照を所有者なしリソースとして認可する", func(t *testing.T) {
			t.Parallel()

			u, deps := newShippableTestUsecase(t)
			var capturedAction authz.Action
			var capturedResource *authz.Resource
			deps.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ *auth.Authn, action authz.Action, resource *authz.Resource) error {
					capturedAction = action
					capturedResource = resource
					return nil
				})
			deps.repo.EXPECT().FindShippable(gomock.Any(), gomock.Any()).Return(domainpurchase.Purchases{}, nil)

			_, err := u.ListShippablePurchases(context.Background(), &auth.Authn{}, ListShippablePurchasesParams{})
			require.NoError(t, err)
			assert.Equal(t, authz.ActionPurchaseListShippable, capturedAction)
			require.NotNil(t, capturedResource)
			assert.Equal(t, "purchase", capturedResource.Kind())
			assert.Nil(t, capturedResource.OwnerID())
		})

		t.Run("limit未指定の場合、既定件数がリポジトリへ渡る", func(t *testing.T) {
			t.Parallel()

			u, deps := newShippableTestUsecase(t)
			deps.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			var capturedLimit int32
			deps.repo.EXPECT().FindShippable(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, limit int32) (domainpurchase.Purchases, error) {
					capturedLimit = limit
					return domainpurchase.Purchases{}, nil
				})

			_, err := u.ListShippablePurchases(
				context.Background(), &auth.Authn{}, ListShippablePurchasesParams{Limit: 0},
			)
			require.NoError(t, err)
			assert.Equal(t, int32(shippableDefaultLimit), capturedLimit)
		})

		t.Run("limitが上限超過の場合、上限へクランプした件数がリポジトリへ渡る", func(t *testing.T) {
			t.Parallel()

			u, deps := newShippableTestUsecase(t)
			deps.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			var capturedLimit int32
			deps.repo.EXPECT().FindShippable(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, limit int32) (domainpurchase.Purchases, error) {
					capturedLimit = limit
					return domainpurchase.Purchases{}, nil
				})

			_, err := u.ListShippablePurchases(
				context.Background(), &auth.Authn{}, ListShippablePurchasesParams{Limit: 1000},
			)
			require.NoError(t, err)
			assert.Equal(t, int32(shippableMaxLimit), capturedLimit)
		})

		t.Run("発送待ちの購入が無い場合、空の一覧を返す", func(t *testing.T) {
			t.Parallel()

			u, deps := newShippableTestUsecase(t)
			deps.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			deps.repo.EXPECT().FindShippable(gomock.Any(), gomock.Any()).Return(domainpurchase.Purchases{}, nil)

			actual, err := u.ListShippablePurchases(context.Background(), &auth.Authn{}, ListShippablePurchasesParams{})
			require.NoError(t, err)
			assert.Empty(t, actual.Groups)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証情報がnilの場合、ErrUnauthenticatedを返し認可もリポジトリも呼ばない", func(t *testing.T) {
			t.Parallel()

			u, deps := newShippableTestUsecase(t)
			deps.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			deps.repo.EXPECT().FindShippable(gomock.Any(), gomock.Any()).Times(0)

			actual, err := u.ListShippablePurchases(context.Background(), nil, ListShippablePurchasesParams{})
			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
			assert.Empty(t, actual.Groups)
		})

		t.Run("認可が拒否された場合、ErrForbiddenを伝播しリポジトリを呼ばない", func(t *testing.T) {
			t.Parallel()

			u, deps := newShippableTestUsecase(t)
			deps.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(authz.ErrForbidden)
			deps.repo.EXPECT().FindShippable(gomock.Any(), gomock.Any()).Times(0)

			actual, err := u.ListShippablePurchases(context.Background(), &auth.Authn{}, ListShippablePurchasesParams{})
			require.ErrorIs(t, err, authz.ErrForbidden)
			assert.Empty(t, actual.Groups)
		})

		t.Run("リポジトリがエラーを返した場合、そのまま伝播する", func(t *testing.T) {
			t.Parallel()

			u, deps := newShippableTestUsecase(t)
			deps.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			deps.repo.EXPECT().FindShippable(gomock.Any(), gomock.Any()).
				Return(nil, apperror.ErrInternal)

			actual, err := u.ListShippablePurchases(context.Background(), &auth.Authn{}, ListShippablePurchasesParams{})
			require.ErrorIs(t, err, apperror.ErrInternal)
			assert.Empty(t, actual.Groups)
		})

		t.Run("発送可能でない購入が混じっていた場合、SQLと述語の乖離としてエラーにする", func(t *testing.T) {
			t.Parallel()

			u, deps := newShippableTestUsecase(t)
			alice := uuidtestkit.NewTestFromSalt(t, "uc_mismatch_alice")
			shipped := newPurchaseWithStatus(
				t, "uc_mismatch", alice, shippableBaseTime, domainpurchase.StatusShipped,
			)

			deps.authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			deps.repo.EXPECT().FindShippable(gomock.Any(), gomock.Any()).
				Return(domainpurchase.Purchases{shipped}, nil)

			actual, err := u.ListShippablePurchases(context.Background(), &auth.Authn{}, ListShippablePurchasesParams{})
			require.ErrorIs(t, err, errNotShippableInShippableRead)
			assert.Empty(t, actual.Groups)
		})
	})
}
