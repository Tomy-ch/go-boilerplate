package cart

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/cart"
	mock_cart "go-boilerplate/internal/domain/cart/mock"
	"go-boilerplate/internal/domain/product"
	mock_product "go-boilerplate/internal/domain/product/mock"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	clocktestkit "go-boilerplate/internal/usecase/boundary/clock/testkit"
	"go-boilerplate/internal/usecase/boundary/tx"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// newPassthroughTx は、本体をそのまま実行するトランザクションマネージャを返します。
func newPassthroughTx(t *testing.T) tx.Manager {
	t.Helper()
	ctrl := gomock.NewController(t)
	m := mock_tx.NewMockManager(ctrl)
	m.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) },
	).AnyTimes()
	return m
}

func newGetUsecase(
	t *testing.T, txm tx.Manager, cartRepo cart.Repository, productRepo product.Repository, clk clock.Clock,
) *usecase {
	t.Helper()
	return &usecase{
		tracer:      observability.NewNoopTracerFactory(t).Usecase(),
		txm:         txm,
		cartRepo:    cartRepo,
		productRepo: productRepo,
		clock:       clk,
	}
}

func Test_usecase_GetCart(t *testing.T) {
	t.Parallel()

	var (
		userID    = uuidtestkit.NewTestFromSalt(t, "cart_get_user")
		productID = uuidtestkit.NewTestFromSalt(t, "cart_get_product")
		now       = time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("再評価した明細と購入可能分の小計を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			seen := newPrice(t, "10.00")
			c := newGuestCart(t, now.Add(time.Hour), newTestCartItem(t, "cart_get_item", productID, 2, &seen))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(c, nil)
			cartRepo.EXPECT().Update(gomock.Any(), c).Return(nil)

			productRepo := mock_product.NewMockRepository(ctrl)
			productRepo.EXPECT().FindByIDs(gomock.Any(), []uuid.UUID{productID}).
				Return(product.Products{newTestProduct(t, productID, "12.50", 5)}, nil)

			uc := newGetUsecase(t, newPassthroughTx(t), cartRepo, productRepo, clocktestkit.NewMockClock(t, now))

			actual, err := uc.GetCart(t.Context(), Subject{UserID: &userID})
			require.NoError(t, err)

			require.Len(t, actual.Items, 1)
			assert.Equal(t, []ItemIssue{ItemIssuePriceIncreased}, actual.Items[0].Issues)
			// 値上がりした明細は購入可能ではないため小計に入らない。
			assert.Zero(t, actual.SubtotalAmount)
			require.NotNil(t, actual.ExpiresAt)
			assert.Equal(t, now.Add(cartTTL), *actual.ExpiresAt)
			// GET はカートを作らないため、セッショントークンを新規発行することがない。
			assert.Nil(t, actual.SessionToken)
		})

		t.Run("提示価格を記録してから返すため2回目は値上がりが立たない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			seen := newPrice(t, "10.00")
			c := newGuestCart(t, now.Add(time.Hour), newTestCartItem(t, "cart_get_marked", productID, 1, &seen))
			p := newTestProduct(t, productID, "12.50", 5)

			var saved *cart.Cart
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(c, nil)
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, updated *cart.Cart) error {
					saved = updated
					return nil
				})

			productRepo := mock_product.NewMockRepository(ctrl)
			productRepo.EXPECT().FindByIDs(gomock.Any(), gomock.Any()).Return(product.Products{p}, nil)

			uc := newGetUsecase(t, newPassthroughTx(t), cartRepo, productRepo, clocktestkit.NewMockClock(t, now))

			_, err := uc.GetCart(t.Context(), Subject{UserID: &userID})
			require.NoError(t, err)

			require.NotNil(t, saved)
			items := saved.Items()
			require.Len(t, items, 1)
			require.NotNil(t, items[0].LastSeenPrice())
			assert.Equal(t, "12.5", items[0].LastSeenPrice().String())
		})

		t.Run("有効期限を延長して永続化する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			c := newGuestCart(t, now.Add(time.Minute))

			var saved *cart.Cart
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(c, nil)
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, updated *cart.Cart) error {
					saved = updated
					return nil
				})

			productRepo := mock_product.NewMockRepository(ctrl)
			productRepo.EXPECT().FindByIDs(gomock.Any(), gomock.Any()).Times(0)

			uc := newGetUsecase(t, newPassthroughTx(t), cartRepo, productRepo, clocktestkit.NewMockClock(t, now))

			actual, err := uc.GetCart(t.Context(), Subject{UserID: &userID})
			require.NoError(t, err)

			require.NotNil(t, saved)
			assert.Equal(t, now.Add(cartTTL), saved.ExpiresAt())
			assert.Empty(t, actual.Items)
		})

		t.Run("主体を持たない場合は空のカートを返し行を作らない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), gomock.Any()).Times(0)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Times(0)
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

			uc := newGetUsecase(t, newPassthroughTx(t), cartRepo,
				mock_product.NewMockRepository(ctrl), clocktestkit.NewMockClock(t, now))

			actual, err := uc.GetCart(t.Context(), Subject{})
			require.NoError(t, err)

			assert.NotNil(t, actual.Items)
			assert.Empty(t, actual.Items)
			assert.Nil(t, actual.ExpiresAt)
		})

		t.Run("カートが無い場合は空のカートを返し行を作らない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).
				Return(nil, xerrors.Wrap(apperror.ErrNotFound, "cart not found"))
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

			uc := newGetUsecase(t, newPassthroughTx(t), cartRepo,
				mock_product.NewMockRepository(ctrl), clocktestkit.NewMockClock(t, now))

			actual, err := uc.GetCart(t.Context(), Subject{UserID: &userID})
			require.NoError(t, err)

			assert.Empty(t, actual.Items)
			assert.Nil(t, actual.ExpiresAt)
		})

		t.Run("有効期限を過ぎたカートには空のカートを返し延長しない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(newGuestCart(t, now.Add(-time.Hour)), nil)
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

			uc := newGetUsecase(t, newPassthroughTx(t), cartRepo,
				mock_product.NewMockRepository(ctrl), clocktestkit.NewMockClock(t, now))

			actual, err := uc.GetCart(t.Context(), Subject{UserID: &userID})
			require.NoError(t, err)

			assert.Empty(t, actual.Items)
			assert.Nil(t, actual.ExpiresAt)
		})

		t.Run("ゲスト主体はセッショントークンでカートを引く", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			token, err := cart.NewSessionToken(testSessionToken)
			require.NoError(t, err)

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), token).Return(newGuestCart(t, now.Add(time.Hour)), nil)
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

			uc := newGetUsecase(t, newPassthroughTx(t), cartRepo,
				mock_product.NewMockRepository(ctrl), clocktestkit.NewMockClock(t, now))

			actual, err := uc.GetCart(t.Context(), Subject{SessionToken: ptr.To(testSessionToken)})
			require.NoError(t, err)

			assert.NotNil(t, actual.ExpiresAt)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("カートの取得に失敗した場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			expectedErr := xerrors.New("connection refused")

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(nil, expectedErr)
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

			uc := newGetUsecase(t, newPassthroughTx(t), cartRepo,
				mock_product.NewMockRepository(ctrl), clocktestkit.NewMockClock(t, now))

			_, err := uc.GetCart(t.Context(), Subject{UserID: &userID})

			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("商品の取得に失敗した場合はカートを更新しない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			expectedErr := xerrors.New("failed to find products")
			c := newGuestCart(t, now.Add(time.Hour), newTestCartItem(t, "cart_get_perr", productID, 1, nil))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(c, nil)
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

			productRepo := mock_product.NewMockRepository(ctrl)
			productRepo.EXPECT().FindByIDs(gomock.Any(), gomock.Any()).Return(nil, expectedErr)

			uc := newGetUsecase(t, newPassthroughTx(t), cartRepo, productRepo, clocktestkit.NewMockClock(t, now))

			_, err := uc.GetCart(t.Context(), Subject{UserID: &userID})

			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("カートの更新に失敗した場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			expectedErr := xerrors.New("failed to update cart")

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(newGuestCart(t, now.Add(time.Hour)), nil)
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(expectedErr)

			uc := newGetUsecase(t, newPassthroughTx(t), cartRepo,
				mock_product.NewMockRepository(ctrl), clocktestkit.NewMockClock(t, now))

			_, err := uc.GetCart(t.Context(), Subject{UserID: &userID})

			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("トランザクションの開始に失敗した場合は本体を実行しない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			expectedErr := xerrors.New("failed to begin tx")

			txm := mock_tx.NewMockManager(ctrl)
			txm.EXPECT().Do(gomock.Any(), gomock.Any()).Return(expectedErr)

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), gomock.Any()).Times(0)

			uc := newGetUsecase(t, txm, cartRepo,
				mock_product.NewMockRepository(ctrl), clocktestkit.NewMockClock(t, now))

			_, err := uc.GetCart(t.Context(), Subject{UserID: &userID})

			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("明細が積み上がり小計が決済スケールへ落とせない場合はErrSubtotalOutOfRangeを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			otherID := uuidtestkit.NewTestFromSalt(t, "cart_get_overflow_other")
			c := newGuestCart(t, now.Add(time.Hour),
				newTestCartItem(t, "cart_get_overflow_a", productID, 1, nil),
				newTestCartItem(t, "cart_get_overflow_b", otherID, 1, nil),
			)

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(c, nil)
			// 合算に失敗する経路では書き込みへ進まない。
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

			productRepo := mock_product.NewMockRepository(ctrl)
			productRepo.EXPECT().FindByIDs(gomock.Any(), gomock.Any()).Return(product.Products{
				newTestProduct(t, productID, "92233720368547758.07", 5),
				newTestProduct(t, otherID, "92233720368547758.07", 5),
			}, nil)

			uc := newGetUsecase(t, newPassthroughTx(t), cartRepo, productRepo, clocktestkit.NewMockClock(t, now))

			_, err := uc.GetCart(t.Context(), Subject{UserID: &userID})

			require.ErrorIs(t, err, cart.ErrSubtotalOutOfRange)
			require.ErrorIs(t, err, apperror.ErrValidation)
			require.ErrorIs(t, err, decimal.ErrOverflow)
		})

		t.Run("セッショントークンの形式が不正ならエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Times(0)

			uc := newGetUsecase(t, newPassthroughTx(t), cartRepo,
				mock_product.NewMockRepository(ctrl), clocktestkit.NewMockClock(t, now))

			_, err := uc.GetCart(t.Context(), Subject{SessionToken: ptr.To("too-short")})

			require.ErrorIs(t, err, cart.ErrInvalidSessionToken)
		})
	})
}
