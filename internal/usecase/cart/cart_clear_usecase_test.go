package cart

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/cart"
	mock_cart "go-boilerplate/internal/domain/cart/mock"
	mock_product "go-boilerplate/internal/domain/product/mock"
	clocktestkit "go-boilerplate/internal/usecase/boundary/clock/testkit"
	"go-boilerplate/internal/usecase/boundary/tx"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// newClearUsecase は、カートのクリアに必要な依存を揃えたユースケースを組み立てます。
// productRepo は期待を持たないモックで、再評価を行わないことがそのまま検証されます。
func newClearUsecase(
	t *testing.T, txm tx.Manager, cartRepo cart.Repository, now time.Time,
) *usecase {
	t.Helper()
	u := newTestUsecase(t, cartRepo, mock_product.NewMockRepository(gomock.NewController(t)))
	u.txm = txm
	u.clock = clocktestkit.NewMockClock(t, now)
	return u
}

func Test_usecase_ClearCart(t *testing.T) {
	t.Parallel()

	var (
		userID      = uuidtestkit.NewTestFromSalt(t, "clear_cart_user")
		productID   = uuidtestkit.NewTestFromSalt(t, "clear_cart_product")
		otherProd   = uuidtestkit.NewTestFromSalt(t, "clear_cart_other")
		now         = time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
		expiresSoon = now.Add(time.Hour)
	)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("明細がすべて取り除かれる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			locked := newOwnerCart(t, userID, expiresSoon,
				newTestCartItem(t, "clear_a", productID, 2, nil),
				newTestCartItem(t, "clear_b", otherProd, 3, nil),
			)

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).
				Return(newOwnerCart(t, userID, expiresSoon), nil)
			cartRepo.EXPECT().LockByID(gomock.Any(), locked.ID()).Return(locked, nil)

			var saved *cart.Cart
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, c *cart.Cart) error {
					saved = c
					return nil
				})

			u := newClearUsecase(t, newPassthroughTx(t), cartRepo, now)

			err := u.ClearCart(context.Background(), Subject{UserID: &userID})
			require.NoError(t, err)

			require.NotNil(t, saved)
			assert.Empty(t, saved.Items())
		})

		t.Run("カートの行と有効期限は残る", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			locked := newOwnerCart(t, userID, expiresSoon,
				newTestCartItem(t, "clear_keep_row", productID, 1, nil),
			)

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).
				Return(newOwnerCart(t, userID, expiresSoon), nil)
			cartRepo.EXPECT().LockByID(gomock.Any(), locked.ID()).Return(locked, nil)

			var saved *cart.Cart
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, c *cart.Cart) error {
					saved = c
					return nil
				})

			u := newClearUsecase(t, newPassthroughTx(t), cartRepo, now)

			err := u.ClearCart(context.Background(), Subject{UserID: &userID})
			require.NoError(t, err)

			require.NotNil(t, saved)
			assert.Equal(t, locked.ID(), saved.ID())
			assert.Equal(t, now.Add(cartTTL), saved.ExpiresAt())
		})

		t.Run("既に空のカートでも成功し有効期限だけが延びる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			locked := newOwnerCart(t, userID, expiresSoon)

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).
				Return(newOwnerCart(t, userID, expiresSoon), nil)
			cartRepo.EXPECT().LockByID(gomock.Any(), locked.ID()).Return(locked, nil)

			var saved *cart.Cart
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, c *cart.Cart) error {
					saved = c
					return nil
				})

			u := newClearUsecase(t, newPassthroughTx(t), cartRepo, now)

			err := u.ClearCart(context.Background(), Subject{UserID: &userID})
			require.NoError(t, err)

			require.NotNil(t, saved)
			assert.Empty(t, saved.Items())
			assert.Equal(t, now.Add(cartTTL), saved.ExpiresAt())
		})

		t.Run("ゲストのカートも空にできる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			locked := newGuestCart(t, expiresSoon,
				newTestCartItem(t, "clear_guest", productID, 1, nil),
			)

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).
				Return(newGuestCart(t, expiresSoon), nil)
			cartRepo.EXPECT().LockByID(gomock.Any(), locked.ID()).Return(locked, nil)

			var saved *cart.Cart
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, c *cart.Cart) error {
					saved = c
					return nil
				})

			u := newClearUsecase(t, newPassthroughTx(t), cartRepo, now)

			token := testSessionToken
			err := u.ClearCart(context.Background(), Subject{SessionToken: &token})
			require.NoError(t, err)

			require.NotNil(t, saved)
			assert.Empty(t, saved.Items())
		})

		t.Run("カートを持たない主体にはカートを作らない", func(t *testing.T) {
			t.Parallel()

			// LockByID / Update / Create の期待を置かないため、触れば gomock が失敗させる。
			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).
				Return(nil, xerrors.Wrap(apperror.ErrNotFound, "cart not found"))

			u := newClearUsecase(t, newPassthroughTx(t), cartRepo, now)

			err := u.ClearCart(context.Background(), Subject{UserID: &userID})

			require.NoError(t, err)
		})

		t.Run("有効期限を過ぎたカートは無いものとして扱う", func(t *testing.T) {
			t.Parallel()

			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).
				Return(newOwnerCart(t, userID, now.Add(-time.Hour)), nil)

			u := newClearUsecase(t, newPassthroughTx(t), cartRepo, now)

			err := u.ClearCart(context.Background(), Subject{UserID: &userID})

			require.NoError(t, err)
		})

		t.Run("主体を持たない呼び出しも成功する", func(t *testing.T) {
			t.Parallel()

			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))

			u := newClearUsecase(t, newPassthroughTx(t), cartRepo, now)

			err := u.ClearCart(context.Background(), Subject{})

			require.NoError(t, err)
		})

		t.Run("処理はトランザクションの内側で行う", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).
				Return(nil, xerrors.Wrap(apperror.ErrNotFound, "cart not found"))

			// 素通しではなく回数を固定する。tx を経由しない実装への退行はここでしか捕まらない。
			txm := mock_tx.NewMockManager(ctrl)
			txm.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) },
			).Times(1)

			u := newClearUsecase(t, txm, cartRepo, now)

			err := u.ClearCart(context.Background(), Subject{UserID: &userID})

			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("カートの取得に失敗した場合はそのまま返す", func(t *testing.T) {
			t.Parallel()

			expected := xerrors.New("find failed")
			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(nil, expected)

			u := newClearUsecase(t, newPassthroughTx(t), cartRepo, now)

			err := u.ClearCart(context.Background(), Subject{UserID: &userID})

			require.ErrorIs(t, err, expected)
		})

		t.Run("解決とロックの間にカートが消えていても保存せず成功する", func(t *testing.T) {
			t.Parallel()

			found := newOwnerCart(t, userID, expiresSoon)

			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(found, nil)
			cartRepo.EXPECT().LockByID(gomock.Any(), found.ID()).
				Return(nil, xerrors.Wrap(apperror.ErrNotFound, "cart not found"))

			u := newClearUsecase(t, newPassthroughTx(t), cartRepo, now)

			err := u.ClearCart(context.Background(), Subject{UserID: &userID})

			require.NoError(t, err)
		})

		t.Run("ロックに失敗した場合は保存しない", func(t *testing.T) {
			t.Parallel()

			expected := xerrors.New("lock failed")
			found := newOwnerCart(t, userID, expiresSoon)

			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(found, nil)
			cartRepo.EXPECT().LockByID(gomock.Any(), found.ID()).Return(nil, expected)

			u := newClearUsecase(t, newPassthroughTx(t), cartRepo, now)

			err := u.ClearCart(context.Background(), Subject{UserID: &userID})

			require.ErrorIs(t, err, expected)
		})

		t.Run("保存に失敗した場合はそのまま返す", func(t *testing.T) {
			t.Parallel()

			expected := xerrors.New("update failed")
			locked := newOwnerCart(t, userID, expiresSoon,
				newTestCartItem(t, "clear_update_fail", productID, 1, nil),
			)

			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).
				Return(newOwnerCart(t, userID, expiresSoon), nil)
			cartRepo.EXPECT().LockByID(gomock.Any(), locked.ID()).Return(locked, nil)
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(expected)

			u := newClearUsecase(t, newPassthroughTx(t), cartRepo, now)

			err := u.ClearCart(context.Background(), Subject{UserID: &userID})

			require.ErrorIs(t, err, expected)
		})

		t.Run("トランザクションの開始に失敗した場合は本体を実行しない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			expected := xerrors.New("failed to begin transaction")

			txm := mock_tx.NewMockManager(ctrl)
			txm.EXPECT().Do(gomock.Any(), gomock.Any()).Return(expected)

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), gomock.Any()).Times(0)

			u := newClearUsecase(t, txm, cartRepo, now)

			err := u.ClearCart(context.Background(), Subject{UserID: &userID})

			require.ErrorIs(t, err, expected)
		})

		t.Run("セッショントークンの形式が不正な場合はそのまま返す", func(t *testing.T) {
			t.Parallel()

			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))

			u := newClearUsecase(t, newPassthroughTx(t), cartRepo, now)

			invalid := "short"
			err := u.ClearCart(context.Background(), Subject{SessionToken: &invalid})

			require.ErrorIs(t, err, cart.ErrInvalidSessionToken)
		})
	})
}
