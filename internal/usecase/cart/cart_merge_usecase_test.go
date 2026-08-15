package cart

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/cart"
	mock_cart "go-boilerplate/internal/domain/cart/mock"
	mock_product "go-boilerplate/internal/domain/product/mock"
	clocktestkit "go-boilerplate/internal/usecase/boundary/clock/testkit"
	"go-boilerplate/internal/usecase/boundary/tx"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// newMergeUsecase は、引き継ぎに必要な依存を揃えたユースケースを組み立てます。
// productRepo は期待を持たないモックで、再評価を行わないことがそのまま検証されます。
func newMergeUsecase(
	t *testing.T, txm tx.Manager, cartRepo cart.Repository, now time.Time,
) *usecase {
	t.Helper()
	u := newTestUsecase(t, cartRepo, mock_product.NewMockRepository(gomock.NewController(t)))
	u.txm = txm
	u.clock = clocktestkit.NewMockClock(t, now)
	return u
}

// expectEnsureOwner は、引き継ぎ先の確保の期待を置きます。
func expectEnsureOwner(cartRepo *mock_cart.MockRepository, owner *cart.Cart) {
	cartRepo.EXPECT().CreateOwnerIfAbsent(gomock.Any(), gomock.Any()).Return(owner, nil)
}

// newFilledItems は、上限まで明細を埋めるための互いに異なる商品の明細を n 件作ります。
func newFilledItems(t *testing.T, salt string, n int) []cart.CartItem {
	t.Helper()
	items := make([]cart.CartItem, 0, n)
	for i := range n {
		s := fmt.Sprintf("%s_%d", salt, i)
		items = append(items, newTestCartItem(t, s, uuidtestkit.NewTestFromSalt(t, s+"_p"), 1, nil))
	}
	return items
}

func Test_usecase_MergeOnLogin(t *testing.T) {
	t.Parallel()

	var (
		userID      = uuidtestkit.NewTestFromSalt(t, "merge_user")
		productID   = uuidtestkit.NewTestFromSalt(t, "merge_product")
		now         = time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
		expiresSoon = now.Add(time.Hour)
	)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ゲストの明細が引き継ぎ先へ移りゲストカートは消える", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			guest := newGuestCart(t, expiresSoon, newTestCartItem(t, "merge_g", productID, 2, nil))
			owner := newOwnerCart(t, userID, expiresSoon)

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Return(guest, nil)
			expectEnsureOwner(cartRepo, owner)
			cartRepo.EXPECT().LockByIDs(gomock.Any(), gomock.Any()).Return(cart.Carts{guest, owner}, nil)

			var saved *cart.Cart
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, c *cart.Cart) error {
					saved = c
					return nil
				})
			cartRepo.EXPECT().Delete(gomock.Any(), guest.ID()).Return(nil)

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			actual, err := u.MergeOnLogin(context.Background(), MergeOnLoginParams{
				UserID:       userID,
				SessionToken: testSessionToken,
			})
			require.NoError(t, err)

			assert.Empty(t, actual.Clamped)
			assert.Empty(t, actual.Dropped)
			require.NotNil(t, saved)
			require.Len(t, saved.Items(), 1)
			assert.Equal(t, productID, saved.Items()[0].ProductID())
		})

		t.Run("同一商品は数量が合算される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			guest := newGuestCart(t, expiresSoon, newTestCartItem(t, "merge_sum_g", productID, 2, nil))
			owner := newOwnerCart(t, userID, expiresSoon,
				newTestCartItem(t, "merge_sum_o", productID, 3, nil))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Return(guest, nil)
			expectEnsureOwner(cartRepo, owner)
			cartRepo.EXPECT().LockByIDs(gomock.Any(), gomock.Any()).Return(cart.Carts{guest, owner}, nil)

			var saved *cart.Cart
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, c *cart.Cart) error {
					saved = c
					return nil
				})
			cartRepo.EXPECT().Delete(gomock.Any(), guest.ID()).Return(nil)

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			_, err := u.MergeOnLogin(context.Background(), MergeOnLoginParams{
				UserID:       userID,
				SessionToken: testSessionToken,
			})
			require.NoError(t, err)

			require.NotNil(t, saved)
			require.Len(t, saved.Items(), 1)
			assert.Equal(t, 5, saved.Items()[0].Quantity())
		})

		t.Run("上限を超える合算はクランプされ報告に載る", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			guest := newGuestCart(t, expiresSoon, newTestCartItem(t, "merge_cl_g", productID, 90, nil))
			owner := newOwnerCart(t, userID, expiresSoon,
				newTestCartItem(t, "merge_cl_o", productID, 50, nil))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Return(guest, nil)
			expectEnsureOwner(cartRepo, owner)
			cartRepo.EXPECT().LockByIDs(gomock.Any(), gomock.Any()).Return(cart.Carts{guest, owner}, nil)
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
			cartRepo.EXPECT().Delete(gomock.Any(), guest.ID()).Return(nil)

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			actual, err := u.MergeOnLogin(context.Background(), MergeOnLoginParams{
				UserID:       userID,
				SessionToken: testSessionToken,
			})
			require.NoError(t, err)

			assert.Equal(t, []uuid.UUID{productID}, actual.Clamped)
			assert.Empty(t, actual.Dropped)
		})

		t.Run("明細数の上限を超える分は切り捨てられ報告に載る", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			guest := newGuestCart(t, expiresSoon, newTestCartItem(t, "merge_dr_g", productID, 1, nil))
			owner := newOwnerCart(t, userID, expiresSoon, newFilledItems(t, "merge_dr_o", 50)...)

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Return(guest, nil)
			expectEnsureOwner(cartRepo, owner)
			cartRepo.EXPECT().LockByIDs(gomock.Any(), gomock.Any()).Return(cart.Carts{guest, owner}, nil)
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
			cartRepo.EXPECT().Delete(gomock.Any(), guest.ID()).Return(nil)

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			actual, err := u.MergeOnLogin(context.Background(), MergeOnLoginParams{
				UserID:       userID,
				SessionToken: testSessionToken,
			})
			require.NoError(t, err)

			assert.Equal(t, []uuid.UUID{productID}, actual.Dropped)
			assert.Empty(t, actual.Clamped)
		})

		t.Run("ロックへ渡すのは引き継ぎ元と引き継ぎ先のIDになる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			guest := newGuestCart(t, expiresSoon)
			owner := newOwnerCart(t, userID, expiresSoon)

			var locked []uuid.UUID
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Return(guest, nil)
			expectEnsureOwner(cartRepo, owner)
			cartRepo.EXPECT().LockByIDs(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, ids []uuid.UUID) (cart.Carts, error) {
					locked = ids
					return cart.Carts{guest, owner}, nil
				})
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
			cartRepo.EXPECT().Delete(gomock.Any(), guest.ID()).Return(nil)

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			_, err := u.MergeOnLogin(context.Background(), MergeOnLoginParams{
				UserID:       userID,
				SessionToken: testSessionToken,
			})
			require.NoError(t, err)

			assert.Equal(t, []uuid.UUID{guest.ID(), owner.ID()}, locked)
		})

		t.Run("引き継ぐカートが無ければ何もせず空の報告を返す", func(t *testing.T) {
			t.Parallel()

			// LockByIDs / Update / Delete の期待を置かないため、触れば gomock が失敗させる。
			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).
				Return(nil, xerrors.Wrap(apperror.ErrNotFound, "cart not found"))

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			actual, err := u.MergeOnLogin(context.Background(), MergeOnLoginParams{
				UserID:       userID,
				SessionToken: testSessionToken,
			})
			require.NoError(t, err)

			assert.Empty(t, actual.Clamped)
			assert.Empty(t, actual.Dropped)
		})

		t.Run("期限切れのゲストカートは引き継がない", func(t *testing.T) {
			t.Parallel()

			// CreateOwnerIfAbsent / LockByIDs の期待を置かないため、進めば gomock が失敗させる。
			expired := newGuestCart(t, now.Add(-time.Second),
				newTestCartItem(t, "merge_exp", productID, 1, nil))
			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Return(expired, nil)

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			actual, err := u.MergeOnLogin(context.Background(), MergeOnLoginParams{
				UserID:       userID,
				SessionToken: testSessionToken,
			})
			require.NoError(t, err)

			assert.Empty(t, actual.Clamped)
			assert.Empty(t, actual.Dropped)
		})

		t.Run("解決とロックの間にゲストカートが消えても成功する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			guest := newGuestCart(t, expiresSoon, newTestCartItem(t, "merge_gone", productID, 1, nil))
			owner := newOwnerCart(t, userID, expiresSoon)

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Return(guest, nil)
			expectEnsureOwner(cartRepo, owner)
			// ゲストカートがロックの時点で消えている（引き継ぎ先だけが返る）。
			cartRepo.EXPECT().LockByIDs(gomock.Any(), gomock.Any()).Return(cart.Carts{owner}, nil)

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			actual, err := u.MergeOnLogin(context.Background(), MergeOnLoginParams{
				UserID:       userID,
				SessionToken: testSessionToken,
			})
			require.NoError(t, err)

			assert.Empty(t, actual.Clamped)
		})

		t.Run("解決とロックの間に引き継ぎ先が消えても成功する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			guest := newGuestCart(t, expiresSoon, newTestCartItem(t, "merge_ogone", productID, 1, nil))
			owner := newOwnerCart(t, userID, expiresSoon)

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Return(guest, nil)
			expectEnsureOwner(cartRepo, owner)
			// 引き継ぎ先がロックの時点で消えている（引き継ぎ元だけが返る）。
			cartRepo.EXPECT().LockByIDs(gomock.Any(), gomock.Any()).Return(cart.Carts{guest}, nil)

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			actual, err := u.MergeOnLogin(context.Background(), MergeOnLoginParams{
				UserID:       userID,
				SessionToken: testSessionToken,
			})
			require.NoError(t, err)

			assert.Empty(t, actual.Clamped)
			assert.Empty(t, actual.Dropped)
		})

		t.Run("処理はトランザクションの内側で1回だけ行う", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).
				Return(nil, xerrors.Wrap(apperror.ErrNotFound, "cart not found"))

			txm := mock_tx.NewMockManager(ctrl)
			txm.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) },
			).Times(1)

			u := newMergeUsecase(t, txm, cartRepo, now)

			_, err := u.MergeOnLogin(context.Background(), MergeOnLoginParams{
				UserID:       userID,
				SessionToken: testSessionToken,
			})

			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("セッショントークンの形式が不正な場合はそのまま返す", func(t *testing.T) {
			t.Parallel()

			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			_, err := u.MergeOnLogin(context.Background(), MergeOnLoginParams{
				UserID:       userID,
				SessionToken: "short",
			})

			require.ErrorIs(t, err, cart.ErrInvalidSessionToken)
		})

		t.Run("引き継ぎ先の確保に失敗した場合はロックへ進まない", func(t *testing.T) {
			t.Parallel()

			expected := xerrors.New("create failed")
			guest := newGuestCart(t, expiresSoon)

			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Return(guest, nil)
			cartRepo.EXPECT().CreateOwnerIfAbsent(gomock.Any(), gomock.Any()).Return(nil, expected)

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			_, err := u.MergeOnLogin(context.Background(), MergeOnLoginParams{
				UserID:       userID,
				SessionToken: testSessionToken,
			})

			require.ErrorIs(t, err, expected)
		})

		t.Run("ロックに失敗した場合は保存しない", func(t *testing.T) {
			t.Parallel()

			expected := xerrors.New("lock failed")
			ctrl := gomock.NewController(t)
			guest := newGuestCart(t, expiresSoon)
			owner := newOwnerCart(t, userID, expiresSoon)

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Return(guest, nil)
			expectEnsureOwner(cartRepo, owner)
			cartRepo.EXPECT().LockByIDs(gomock.Any(), gomock.Any()).Return(nil, expected)

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			_, err := u.MergeOnLogin(context.Background(), MergeOnLoginParams{
				UserID:       userID,
				SessionToken: testSessionToken,
			})

			require.ErrorIs(t, err, expected)
		})

		t.Run("トランザクションの開始に失敗した場合は本体を実行しない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			expected := xerrors.New("failed to begin transaction")

			txm := mock_tx.NewMockManager(ctrl)
			txm.EXPECT().Do(gomock.Any(), gomock.Any()).Return(expected)

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Times(0)

			u := newMergeUsecase(t, txm, cartRepo, now)

			_, err := u.MergeOnLogin(context.Background(), MergeOnLoginParams{
				UserID:       userID,
				SessionToken: testSessionToken,
			})

			require.ErrorIs(t, err, expected)
		})
	})
}

func Test_usecase_findGuestCartID(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("トークンからカートのIDを引く", func(t *testing.T) {
			t.Parallel()

			guest := newGuestCart(t, now.Add(time.Hour))
			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Return(guest, nil)

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			id, found, err := u.findGuestCartID(context.Background(), testSessionToken, now)
			require.NoError(t, err)

			assert.True(t, found)
			assert.Equal(t, guest.ID(), id)
		})

		t.Run("引けない場合は失敗ではなくfoundがfalseになる", func(t *testing.T) {
			t.Parallel()

			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).
				Return(nil, xerrors.Wrap(apperror.ErrNotFound, "cart not found"))

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			_, found, err := u.findGuestCartID(context.Background(), testSessionToken, now)
			require.NoError(t, err)

			assert.False(t, found)
		})

		t.Run("期限切れのカートは引き継ぐものが無いのと同じ扱いになる", func(t *testing.T) {
			t.Parallel()

			// 期限切れの行は物理削除されるまで引けてしまうため、ここで落とさないと引き継がれる。
			expired := newGuestCart(t, now.Add(-time.Second))
			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Return(expired, nil)

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			_, found, err := u.findGuestCartID(context.Background(), testSessionToken, now)
			require.NoError(t, err)

			assert.False(t, found)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("トークンの形式が不正ならそのまま返す", func(t *testing.T) {
			t.Parallel()

			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			_, _, err := u.findGuestCartID(context.Background(), "short", now)

			require.ErrorIs(t, err, cart.ErrInvalidSessionToken)
		})

		t.Run("取得に失敗した場合はそのまま返す", func(t *testing.T) {
			t.Parallel()

			expected := xerrors.New("find failed")
			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Return(nil, expected)

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			_, _, err := u.findGuestCartID(context.Background(), testSessionToken, now)

			require.ErrorIs(t, err, expected)
		})
	})
}

func Test_usecase_ensureOwnerCartID(t *testing.T) {
	t.Parallel()

	var (
		userID = uuidtestkit.NewTestFromSalt(t, "ensure_user")
		now    = time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("確保したカートのIDを返す", func(t *testing.T) {
			t.Parallel()

			// 作成が競合しても勝ったほうの行が返るため、渡した候補ではなく返った行の ID になる。
			existing := newOwnerCart(t, userID, now.Add(time.Hour))

			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))
			cartRepo.EXPECT().CreateOwnerIfAbsent(gomock.Any(), gomock.Any()).Return(existing, nil)

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			id, err := u.ensureOwnerCartID(context.Background(), userID, now)
			require.NoError(t, err)

			assert.Equal(t, existing.ID(), id)
		})

		t.Run("所有者と有効期限を持つ候補を渡す", func(t *testing.T) {
			t.Parallel()

			existing := newOwnerCart(t, userID, now.Add(time.Hour))

			var candidate *cart.Cart
			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))
			cartRepo.EXPECT().CreateOwnerIfAbsent(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, c *cart.Cart) (*cart.Cart, error) {
					candidate = c
					return existing, nil
				})

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			_, err := u.ensureOwnerCartID(context.Background(), userID, now)
			require.NoError(t, err)

			require.NotNil(t, candidate)
			require.NotNil(t, candidate.OwnerID())
			assert.Equal(t, userID, *candidate.OwnerID())
			assert.Equal(t, now.Add(cartTTL), candidate.ExpiresAt())
			assert.Nil(t, candidate.SessionToken())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("確保に失敗した場合はそのまま返す", func(t *testing.T) {
			t.Parallel()

			expected := xerrors.New("create failed")
			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))
			cartRepo.EXPECT().CreateOwnerIfAbsent(gomock.Any(), gomock.Any()).Return(nil, expected)

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			_, err := u.ensureOwnerCartID(context.Background(), userID, now)

			require.ErrorIs(t, err, expected)
		})
	})
}

func Test_usecase_mergeInto(t *testing.T) {
	t.Parallel()

	var (
		userID    = uuidtestkit.NewTestFromSalt(t, "into_user")
		productID = uuidtestkit.NewTestFromSalt(t, "into_product")
		now       = time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("取り込んでからゲストカートを消し有効期限を延ばす", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			guest := newGuestCart(t, now.Add(time.Hour), newTestCartItem(t, "into_g", productID, 1, nil))
			owner := newOwnerCart(t, userID, now.Add(time.Hour))

			var saved *cart.Cart
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, c *cart.Cart) error {
					saved = c
					return nil
				})
			cartRepo.EXPECT().Delete(gomock.Any(), guest.ID()).Return(nil)

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			actual, err := u.mergeInto(context.Background(), owner, guest, now)
			require.NoError(t, err)

			assert.Empty(t, actual.Clamped)
			require.NotNil(t, saved)
			assert.Len(t, saved.Items(), 1)
			assert.Equal(t, now.Add(cartTTL), saved.ExpiresAt())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保存に失敗した場合はゲストカートを消さない", func(t *testing.T) {
			t.Parallel()

			expected := xerrors.New("update failed")
			guest := newGuestCart(t, now.Add(time.Hour))
			owner := newOwnerCart(t, userID, now.Add(time.Hour))

			// Delete の期待を置かないため、消せば gomock が失敗させる。
			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(expected)

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			_, err := u.mergeInto(context.Background(), owner, guest, now)

			require.ErrorIs(t, err, expected)
		})

		t.Run("破棄に失敗した場合はそのまま返す", func(t *testing.T) {
			t.Parallel()

			expected := xerrors.New("delete failed")
			guest := newGuestCart(t, now.Add(time.Hour))
			owner := newOwnerCart(t, userID, now.Add(time.Hour))

			cartRepo := mock_cart.NewMockRepository(gomock.NewController(t))
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
			cartRepo.EXPECT().Delete(gomock.Any(), guest.ID()).Return(expected)

			u := newMergeUsecase(t, newPassthroughTx(t), cartRepo, now)

			_, err := u.mergeInto(context.Background(), owner, guest, now)

			require.ErrorIs(t, err, expected)
		})
	})
}

func Test_indexCarts(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("IDで引ける表になる", func(t *testing.T) {
			t.Parallel()

			now := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
			guest := newGuestCart(t, now.Add(time.Hour))
			owner := newOwnerCart(t, uuidtestkit.NewTestFromSalt(t, "idx_user"), now.Add(time.Hour))

			actual := indexCarts(cart.Carts{guest, owner})

			require.Len(t, actual, 2)
			assert.Same(t, guest, actual[guest.ID()])
			assert.Same(t, owner, actual[owner.ID()])
		})

		t.Run("空なら空の表を返す", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, indexCarts(cart.Carts{}))
		})
	})
}
