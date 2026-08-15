package cart

import (
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/cart"
	mock_cart "go-boilerplate/internal/domain/cart/mock"
	"go-boilerplate/internal/domain/product"
	mock_product "go-boilerplate/internal/domain/product/mock"
	clocktestkit "go-boilerplate/internal/usecase/boundary/clock/testkit"
	"go-boilerplate/internal/usecase/boundary/token"
	mock_token "go-boilerplate/internal/usecase/boundary/token/mock"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// issuedSessionToken は、採番し直したときに返す想定の 43 文字トークン。
const issuedSessionToken = "issued-session-token-for-tests-000000000000"

// maxUnitPrice は、money.Price が受理する上限の単価。2 件積むと決済スケールを超える。
const maxUnitPrice = "92233720368547758.07"

// newSetItemUsecase は、明細設定に必要な依存を揃えたユースケースを組み立てます。
func newSetItemUsecase(
	t *testing.T,
	cartRepo cart.Repository,
	productRepo product.Repository,
	tokenGen token.Generator,
	now time.Time,
) *usecase {
	t.Helper()
	u := newTestUsecase(t, cartRepo, productRepo)
	u.txm = newPassthroughTx(t)
	u.tokenGen = tokenGen
	u.clock = clocktestkit.NewMockClock(t, now)
	return u
}

// newOwnerCart は、所有者が確定したカートを組み立てます。
func newOwnerCart(t *testing.T, ownerID uuid.UUID, expiresAt time.Time, items ...cart.CartItem) *cart.Cart {
	t.Helper()
	c, err := cart.Reconstruct(uuidtestkit.NewTestFromSalt(t, "cart_owner"), cart.Attributes{
		OwnerID:   &ownerID,
		Items:     items,
		ExpiresAt: expiresAt,
	})
	require.NoError(t, err)
	return c
}

// newIssuingTokenGen は、決まったトークンを 1 回返す生成器を組み立てます。
func newIssuingTokenGen(t *testing.T) token.Generator {
	t.Helper()
	gen := mock_token.NewMockGenerator(gomock.NewController(t))
	gen.EXPECT().Generate().Return(issuedSessionToken, nil).Times(1)
	return gen
}

func Test_usecase_SetItem(t *testing.T) {
	t.Parallel()

	var (
		userID    = uuidtestkit.NewTestFromSalt(t, "set_item_user")
		productID = uuidtestkit.NewTestFromSalt(t, "set_item_product")
		now       = time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("カートを持たないゲストにはカートとトークンが作られる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

			productRepo := mock_product.NewMockRepository(ctrl)
			productRepo.EXPECT().FindPublishedByID(gomock.Any(), productID).
				Return(newTestProduct(t, productID, "10.00", 5), nil)
			productRepo.EXPECT().FindByIDs(gomock.Any(), []uuid.UUID{productID}).
				Return(product.Products{newTestProduct(t, productID, "10.00", 5)}, nil)

			uc := newSetItemUsecase(t, cartRepo, productRepo, newIssuingTokenGen(t), now)

			actual, err := uc.SetItem(t.Context(), SetItemParams{ProductID: productID, Quantity: 2})
			require.NoError(t, err)

			require.NotNil(t, actual.SessionToken)
			assert.Equal(t, issuedSessionToken, *actual.SessionToken)
			require.Len(t, actual.Items, 1)
			assert.Equal(t, 2, actual.Items[0].Quantity)
			assert.Equal(t, int64(2000), actual.SubtotalAmount)
		})

		t.Run("作成が競合しても、やり直して勝った側のカートへ設定する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			winner := newOwnerCart(t, userID, now.Add(time.Hour))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			gomock.InOrder(
				cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).
					Return(nil, xerrors.Wrap(apperror.ErrNotFound, "not found")),
				cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
					Return(xerrors.Wrap(apperror.ErrConflict, "duplicate key")),
				cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(winner, nil),
				cartRepo.EXPECT().LockByID(gomock.Any(), winner.ID()).Return(winner, nil),
				cartRepo.EXPECT().Update(gomock.Any(), winner).Return(nil),
			)

			productRepo := mock_product.NewMockRepository(ctrl)
			productRepo.EXPECT().FindPublishedByID(gomock.Any(), productID).
				Return(newTestProduct(t, productID, "10.00", 5), nil)
			productRepo.EXPECT().FindByIDs(gomock.Any(), []uuid.UUID{productID}).
				Return(product.Products{newTestProduct(t, productID, "10.00", 5)}, nil)

			uc := newSetItemUsecase(t, cartRepo, productRepo, nil, now)

			actual, err := uc.SetItem(t.Context(), SetItemParams{
				Subject: Subject{UserID: &userID}, ProductID: productID, Quantity: 1,
			})
			require.NoError(t, err)

			// 勝った側のカートを使うため、負けた側が採番したトークンは返らない。
			assert.Nil(t, actual.SessionToken)
			assert.Equal(t, int64(1000), actual.SubtotalAmount)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("やり直しても作成に負け続けるなら衝突として返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).
				Return(nil, xerrors.Wrap(apperror.ErrNotFound, "not found")).Times(maxSetItemAttempts)
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
				Return(xerrors.Wrap(apperror.ErrConflict, "duplicate key")).Times(maxSetItemAttempts)
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

			uc := newSetItemUsecase(t, cartRepo, mock_product.NewMockRepository(ctrl), nil, now)

			_, err := uc.SetItem(t.Context(), SetItemParams{
				Subject: Subject{UserID: &userID}, ProductID: productID, Quantity: 1,
			})

			require.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("作成の衝突でない失敗はやり直さない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).
				Return(nil, xerrors.Wrap(apperror.ErrInternal, "connection reset")).Times(1)

			uc := newSetItemUsecase(t, cartRepo, mock_product.NewMockRepository(ctrl), nil, now)

			_, err := uc.SetItem(t.Context(), SetItemParams{
				Subject: Subject{UserID: &userID}, ProductID: productID, Quantity: 1,
			})

			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("トランザクションの開始に失敗した場合は本体を実行せずやり直しもしない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			expectedErr := xerrors.New("failed to begin transaction")

			// 素通しではなく厳密なモックにする。ちょうど 1 回だけ呼ばれることを固定すると、
			// トランザクションを経ずに本体を呼ぶ実装も、tx の失敗をレースと取り違えて
			// やり直す実装も、どちらも赤くなる。
			txm := mock_tx.NewMockManager(ctrl)
			txm.EXPECT().Do(gomock.Any(), gomock.Any()).Return(expectedErr)

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), gomock.Any()).Times(0)

			uc := newSetItemUsecase(t, cartRepo, mock_product.NewMockRepository(ctrl), nil, now)
			uc.txm = txm

			_, err := uc.SetItem(t.Context(), SetItemParams{
				Subject: Subject{UserID: &userID}, ProductID: productID, Quantity: 1,
			})

			require.ErrorIs(t, err, expectedErr)
		})
	})
}

func Test_usecase_setItem(t *testing.T) {
	t.Parallel()

	var (
		userID    = uuidtestkit.NewTestFromSalt(t, "inner_set_user")
		productID = uuidtestkit.NewTestFromSalt(t, "inner_set_product")
		otherID   = uuidtestkit.NewTestFromSalt(t, "inner_set_other")
		now       = time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同じ要求を2回与えても数量が二重に積まれない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			c := newOwnerCart(t, userID, now.Add(time.Hour))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(c, nil).Times(2)
			cartRepo.EXPECT().LockByID(gomock.Any(), c.ID()).Return(c, nil).Times(2)
			cartRepo.EXPECT().Update(gomock.Any(), c).Return(nil).Times(2)

			productRepo := mock_product.NewMockRepository(ctrl)
			productRepo.EXPECT().FindPublishedByID(gomock.Any(), productID).
				Return(newTestProduct(t, productID, "10.00", 5), nil).Times(2)
			productRepo.EXPECT().FindByIDs(gomock.Any(), []uuid.UUID{productID}).
				Return(product.Products{newTestProduct(t, productID, "10.00", 5)}, nil).Times(2)

			uc := newSetItemUsecase(t, cartRepo, productRepo, nil, now)
			params := SetItemParams{Subject: Subject{UserID: &userID}, ProductID: productID, Quantity: 3}

			_, err := uc.setItem(t.Context(), params)
			require.NoError(t, err)

			actual, err := uc.setItem(t.Context(), params)
			require.NoError(t, err)

			require.Len(t, actual.Items, 1)
			assert.Equal(t, 3, actual.Items[0].Quantity)
		})

		t.Run("既存のカートを使った場合はトークンを返さない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			c := newOwnerCart(t, userID, now.Add(time.Hour))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(c, nil)
			cartRepo.EXPECT().LockByID(gomock.Any(), c.ID()).Return(c, nil)
			cartRepo.EXPECT().Update(gomock.Any(), c).Return(nil)

			productRepo := mock_product.NewMockRepository(ctrl)
			productRepo.EXPECT().FindPublishedByID(gomock.Any(), productID).
				Return(newTestProduct(t, productID, "10.00", 5), nil)
			productRepo.EXPECT().FindByIDs(gomock.Any(), []uuid.UUID{productID}).
				Return(product.Products{newTestProduct(t, productID, "10.00", 5)}, nil)

			uc := newSetItemUsecase(t, cartRepo, productRepo, nil, now)

			actual, err := uc.setItem(t.Context(), SetItemParams{
				Subject: Subject{UserID: &userID}, ProductID: productID, Quantity: 1,
			})
			require.NoError(t, err)

			assert.Nil(t, actual.SessionToken)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("カートへ入れられない商品ならErrUnavailableProductを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			c := newOwnerCart(t, userID, now.Add(time.Hour))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(c, nil)
			cartRepo.EXPECT().LockByID(gomock.Any(), c.ID()).Return(c, nil)
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

			productRepo := mock_product.NewMockRepository(ctrl)
			productRepo.EXPECT().FindPublishedByID(gomock.Any(), productID).
				Return(nil, xerrors.Wrap(apperror.ErrNotFound, "not found"))

			uc := newSetItemUsecase(t, cartRepo, productRepo, nil, now)

			_, err := uc.setItem(t.Context(), SetItemParams{
				Subject: Subject{UserID: &userID}, ProductID: productID, Quantity: 1,
			})

			require.ErrorIs(t, err, ErrUnavailableProduct)
			require.ErrorIs(t, err, apperror.ErrValidation)
		})

		t.Run("数量が範囲外ならドメインのErrInvalidQuantityを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			c := newOwnerCart(t, userID, now.Add(time.Hour))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(c, nil)
			cartRepo.EXPECT().LockByID(gomock.Any(), c.ID()).Return(c, nil)
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

			productRepo := mock_product.NewMockRepository(ctrl)
			productRepo.EXPECT().FindPublishedByID(gomock.Any(), productID).
				Return(newTestProduct(t, productID, "10.00", 5), nil)

			uc := newSetItemUsecase(t, cartRepo, productRepo, nil, now)

			_, err := uc.setItem(t.Context(), SetItemParams{
				Subject: Subject{UserID: &userID}, ProductID: productID, Quantity: 0,
			})

			require.ErrorIs(t, err, cart.ErrInvalidQuantity)
		})

		t.Run("小計が決済スケールを超える明細はカートへ書かれない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			seen := newPrice(t, maxUnitPrice)
			c := newOwnerCart(t, userID, now.Add(time.Hour),
				newTestCartItem(t, "inner_set_existing", otherID, 1, &seen))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(c, nil)
			cartRepo.EXPECT().LockByID(gomock.Any(), c.ID()).Return(c, nil)
			// 書き込みへ進まないことがこの検証の主眼。tx がロールバックするより手前で止まる。
			cartRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

			productRepo := mock_product.NewMockRepository(ctrl)
			productRepo.EXPECT().FindPublishedByID(gomock.Any(), productID).
				Return(newTestProduct(t, productID, maxUnitPrice, 5), nil)
			productRepo.EXPECT().FindByIDs(gomock.Any(), gomock.Any()).Return(product.Products{
				newTestProduct(t, otherID, maxUnitPrice, 5),
				newTestProduct(t, productID, maxUnitPrice, 5),
			}, nil)

			uc := newSetItemUsecase(t, cartRepo, productRepo, nil, now)

			_, err := uc.setItem(t.Context(), SetItemParams{
				Subject: Subject{UserID: &userID}, ProductID: productID, Quantity: 1,
			})

			require.ErrorIs(t, err, cart.ErrSubtotalOutOfRange)
			require.ErrorIs(t, err, decimal.ErrOverflow)
		})

		t.Run("カートの保存に失敗した場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			expectedErr := xerrors.New("connection refused")
			c := newOwnerCart(t, userID, now.Add(time.Hour))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(c, nil)
			cartRepo.EXPECT().LockByID(gomock.Any(), c.ID()).Return(c, nil)
			cartRepo.EXPECT().Update(gomock.Any(), c).Return(expectedErr)

			productRepo := mock_product.NewMockRepository(ctrl)
			productRepo.EXPECT().FindPublishedByID(gomock.Any(), productID).
				Return(newTestProduct(t, productID, "10.00", 5), nil)
			productRepo.EXPECT().FindByIDs(gomock.Any(), []uuid.UUID{productID}).
				Return(product.Products{newTestProduct(t, productID, "10.00", 5)}, nil)

			uc := newSetItemUsecase(t, cartRepo, productRepo, nil, now)

			_, err := uc.setItem(t.Context(), SetItemParams{
				Subject: Subject{UserID: &userID}, ProductID: productID, Quantity: 1,
			})

			require.ErrorIs(t, err, expectedErr)
		})
	})
}

func Test_usecase_ensureProductAvailable(t *testing.T) {
	t.Parallel()

	productID := uuidtestkit.NewTestFromSalt(t, "ensure_product")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("公開中の商品なら通す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			productRepo := mock_product.NewMockRepository(ctrl)
			productRepo.EXPECT().FindPublishedByID(gomock.Any(), productID).
				Return(newTestProduct(t, productID, "10.00", 5), nil)

			err := newTestUsecase(t, nil, productRepo).ensureProductAvailable(t.Context(), productID)
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("引けない商品はErrUnavailableProductになる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			productRepo := mock_product.NewMockRepository(ctrl)
			productRepo.EXPECT().FindPublishedByID(gomock.Any(), productID).
				Return(nil, xerrors.Wrap(apperror.ErrNotFound, "not found"))

			err := newTestUsecase(t, nil, productRepo).ensureProductAvailable(t.Context(), productID)

			require.ErrorIs(t, err, ErrUnavailableProduct)
		})

		t.Run("取得そのものの失敗はそのまま返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			productRepo := mock_product.NewMockRepository(ctrl)
			productRepo.EXPECT().FindPublishedByID(gomock.Any(), productID).
				Return(nil, xerrors.Wrap(apperror.ErrInternal, "connection reset"))

			err := newTestUsecase(t, nil, productRepo).ensureProductAvailable(t.Context(), productID)

			require.ErrorIs(t, err, apperror.ErrInternal)
			require.NotErrorIs(t, err, ErrUnavailableProduct)
		})
	})
}

func Test_usecase_resolveOrCreateCart(t *testing.T) {
	t.Parallel()

	var (
		userID = uuidtestkit.NewTestFromSalt(t, "resolve_or_create_user")
		now    = time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証済みの主体は所有者のカートを引く", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			c := newOwnerCart(t, userID, now.Add(time.Hour))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(c, nil)
			cartRepo.EXPECT().LockByID(gomock.Any(), c.ID()).Return(c, nil)

			uc := newSetItemUsecase(t, cartRepo, nil, nil, now)

			actual, issued, err := uc.resolveOrCreateCart(t.Context(), Subject{UserID: &userID}, now)
			require.NoError(t, err)

			assert.Equal(t, c, actual)
			assert.Nil(t, issued)
		})

		t.Run("主体を持たない呼び出しは新しいゲストカートになる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

			uc := newSetItemUsecase(t, cartRepo, nil, newIssuingTokenGen(t), now)

			actual, issued, err := uc.resolveOrCreateCart(t.Context(), Subject{}, now)
			require.NoError(t, err)

			require.NotNil(t, issued)
			assert.Equal(t, issuedSessionToken, *issued)
			assert.Nil(t, actual.OwnerID())
		})
	})
}

func Test_usecase_resolveOwnerCart(t *testing.T) {
	t.Parallel()

	var (
		userID    = uuidtestkit.NewTestFromSalt(t, "resolve_owner_user")
		productID = uuidtestkit.NewTestFromSalt(t, "resolve_owner_product")
		now       = time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な既存カートはロックして返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			seen := newPrice(t, "10.00")
			c := newOwnerCart(t, userID, now.Add(time.Hour),
				newTestCartItem(t, "resolve_owner_item", productID, 1, &seen))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(c, nil)
			cartRepo.EXPECT().LockByID(gomock.Any(), c.ID()).Return(c, nil)

			actual, issued, err := newSetItemUsecase(t, cartRepo, nil, nil, now).
				resolveOwnerCart(t.Context(), userID, now)
			require.NoError(t, err)

			assert.Len(t, actual.Items(), 1)
			assert.Nil(t, issued)
		})

		t.Run("期限切れのカートは行を保ったまま明細を捨てる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			seen := newPrice(t, "10.00")
			expired := newOwnerCart(t, userID, now.Add(-time.Hour),
				newTestCartItem(t, "resolve_owner_stale", productID, 1, &seen))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(expired, nil)
			cartRepo.EXPECT().LockByID(gomock.Any(), expired.ID()).Return(expired, nil)
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

			actual, _, err := newSetItemUsecase(t, cartRepo, nil, nil, now).
				resolveOwnerCart(t.Context(), userID, now)
			require.NoError(t, err)

			assert.Equal(t, expired.ID(), actual.ID())
			assert.Empty(t, actual.Items())
		})

		t.Run("カートが無ければ作る", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).
				Return(nil, xerrors.Wrap(apperror.ErrNotFound, "not found"))
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

			actual, issued, err := newSetItemUsecase(t, cartRepo, nil, nil, now).
				resolveOwnerCart(t.Context(), userID, now)
			require.NoError(t, err)

			assert.Equal(t, userID, *actual.OwnerID())
			assert.Nil(t, issued)
		})

		t.Run("解決とロックの間にカートが消えていれば作り直す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			c := newOwnerCart(t, userID, now.Add(time.Hour))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(c, nil)
			cartRepo.EXPECT().LockByID(gomock.Any(), c.ID()).
				Return(nil, xerrors.Wrap(apperror.ErrNotFound, "not found"))
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

			actual, _, err := newSetItemUsecase(t, cartRepo, nil, nil, now).
				resolveOwnerCart(t.Context(), userID, now)
			require.NoError(t, err)

			assert.Equal(t, userID, *actual.OwnerID())
			assert.NotEqual(t, c.ID(), actual.ID())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("取得の失敗は作成へ倒さずそのまま返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).
				Return(nil, xerrors.Wrap(apperror.ErrInternal, "connection reset"))
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

			_, _, err := newSetItemUsecase(t, cartRepo, nil, nil, now).
				resolveOwnerCart(t.Context(), userID, now)

			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("ロックの失敗はそのまま返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			c := newOwnerCart(t, userID, now.Add(time.Hour))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(c, nil)
			cartRepo.EXPECT().LockByID(gomock.Any(), c.ID()).
				Return(nil, xerrors.Wrap(apperror.ErrInternal, "deadlock"))

			_, _, err := newSetItemUsecase(t, cartRepo, nil, nil, now).
				resolveOwnerCart(t.Context(), userID, now)

			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}

func Test_usecase_resolveGuestCart(t *testing.T) {
	t.Parallel()

	var (
		productID = uuidtestkit.NewTestFromSalt(t, "resolve_guest_product")
		now       = time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("トークン未提示なら採番して作る", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

			actual, issued, err := newSetItemUsecase(t, cartRepo, nil, newIssuingTokenGen(t), now).
				resolveGuestCart(t.Context(), nil, now)
			require.NoError(t, err)

			require.NotNil(t, issued)
			assert.Equal(t, issuedSessionToken, *issued)
			assert.Equal(t, issuedSessionToken, actual.SessionToken().Value())
		})

		t.Run("有効なトークンならそのカートをロックして返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			seen := newPrice(t, "10.00")
			c := newGuestCart(t, now.Add(time.Hour),
				newTestCartItem(t, "resolve_guest_item", productID, 1, &seen))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Return(c, nil)
			cartRepo.EXPECT().LockByID(gomock.Any(), c.ID()).Return(c, nil)
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

			presented := testSessionToken
			actual, issued, err := newSetItemUsecase(t, cartRepo, nil, nil, now).
				resolveGuestCart(t.Context(), &presented, now)
			require.NoError(t, err)

			assert.Equal(t, c.ID(), actual.ID())
			assert.Nil(t, issued)
		})

		t.Run("未知のトークンは提示値では作らず採番し直す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).
				Return(nil, xerrors.Wrap(apperror.ErrNotFound, "not found"))
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

			presented := testSessionToken
			actual, issued, err := newSetItemUsecase(t, cartRepo, nil, newIssuingTokenGen(t), now).
				resolveGuestCart(t.Context(), &presented, now)
			require.NoError(t, err)

			require.NotNil(t, issued)
			assert.Equal(t, issuedSessionToken, *issued)
			assert.NotEqual(t, presented, actual.SessionToken().Value())
		})

		t.Run("期限切れのカートは復活させず採番し直す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			seen := newPrice(t, "10.00")
			expired := newGuestCart(t, now.Add(-time.Hour),
				newTestCartItem(t, "resolve_guest_stale", productID, 1, &seen))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Return(expired, nil)
			cartRepo.EXPECT().LockByID(gomock.Any(), expired.ID()).Return(expired, nil)
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

			presented := testSessionToken
			actual, issued, err := newSetItemUsecase(t, cartRepo, nil, newIssuingTokenGen(t), now).
				resolveGuestCart(t.Context(), &presented, now)
			require.NoError(t, err)

			assert.NotNil(t, issued)
			assert.NotEqual(t, expired.ID(), actual.ID())
			assert.Empty(t, actual.Items())
		})

		t.Run("解決とロックの間にカートが消えていれば採番し直す", func(t *testing.T) {
			t.Parallel()

			// 引き継ぎ（MergeOnLogin）がゲストカートを行ごと消すため、この窓は実際に開く。
			ctrl := gomock.NewController(t)
			c := newGuestCart(t, now.Add(time.Hour))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Return(c, nil)
			cartRepo.EXPECT().LockByID(gomock.Any(), c.ID()).
				Return(nil, xerrors.Wrap(apperror.ErrNotFound, "not found"))
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

			presented := testSessionToken
			actual, issued, err := newSetItemUsecase(t, cartRepo, nil, newIssuingTokenGen(t), now).
				resolveGuestCart(t.Context(), &presented, now)
			require.NoError(t, err)

			require.NotNil(t, issued)
			assert.Equal(t, issuedSessionToken, *issued)
			assert.NotEqual(t, c.ID(), actual.ID())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("形式が不正なトークンはカートを引かずに弾く", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Times(0)
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

			presented := "too-short"
			_, _, err := newSetItemUsecase(t, cartRepo, nil, nil, now).
				resolveGuestCart(t.Context(), &presented, now)

			require.ErrorIs(t, err, cart.ErrInvalidSessionToken)
		})

		t.Run("取得の失敗は採番へ倒さずそのまま返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).
				Return(nil, xerrors.Wrap(apperror.ErrInternal, "connection reset"))
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

			presented := testSessionToken
			_, _, err := newSetItemUsecase(t, cartRepo, nil, nil, now).
				resolveGuestCart(t.Context(), &presented, now)

			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("ロックの失敗は採番へ倒さずそのまま返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			c := newGuestCart(t, now.Add(time.Hour))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Return(c, nil)
			cartRepo.EXPECT().LockByID(gomock.Any(), c.ID()).
				Return(nil, xerrors.Wrap(apperror.ErrInternal, "deadlock"))
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

			presented := testSessionToken
			_, _, err := newSetItemUsecase(t, cartRepo, nil, nil, now).
				resolveGuestCart(t.Context(), &presented, now)

			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}

func Test_usecase_createOwnerCart(t *testing.T) {
	t.Parallel()

	var (
		userID = uuidtestkit.NewTestFromSalt(t, "create_owner_user")
		now    = time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("所有者と有効期限を持つ空のカートを作る", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

			actual, issued, err := newSetItemUsecase(t, cartRepo, nil, nil, now).
				createOwnerCart(t.Context(), userID, now)
			require.NoError(t, err)

			assert.Equal(t, userID, *actual.OwnerID())
			assert.Equal(t, now.Add(cartTTL), actual.ExpiresAt())
			assert.Empty(t, actual.Items())
			assert.Nil(t, issued)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("一意制約違反はやり直しの対象として印が付く", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
				Return(xerrors.Wrap(apperror.ErrConflict, "duplicate key"))

			_, _, err := newSetItemUsecase(t, cartRepo, nil, nil, now).
				createOwnerCart(t.Context(), userID, now)

			require.ErrorIs(t, err, errCartCreationRace)
		})
	})
}

func Test_usecase_createGuestCart(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("採番したトークンを持つ空のカートを作って返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

			actual, issued, err := newSetItemUsecase(t, cartRepo, nil, newIssuingTokenGen(t), now).
				createGuestCart(t.Context(), now)
			require.NoError(t, err)

			require.NotNil(t, issued)
			assert.Equal(t, issuedSessionToken, *issued)
			assert.Equal(t, issuedSessionToken, actual.SessionToken().Value())
			assert.Nil(t, actual.OwnerID())
			assert.Equal(t, now.Add(cartTTL), actual.ExpiresAt())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("トークンを採番できなければカートを作らない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

			expectedErr := xerrors.New("no entropy")
			gen := mock_token.NewMockGenerator(ctrl)
			gen.EXPECT().Generate().Return("", expectedErr)

			_, _, err := newSetItemUsecase(t, cartRepo, nil, gen, now).createGuestCart(t.Context(), now)

			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("作成が競合したらやり直しの対象として印が付く", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
				Return(xerrors.Wrap(apperror.ErrConflict, "duplicate key"))

			_, _, err := newSetItemUsecase(t, cartRepo, nil, newIssuingTokenGen(t), now).
				createGuestCart(t.Context(), now)

			require.ErrorIs(t, err, errCartCreationRace)
		})

		t.Run("採番した値の形式が不正ならカートを作らない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

			gen := mock_token.NewMockGenerator(ctrl)
			gen.EXPECT().Generate().Return("too-short", nil)

			_, _, err := newSetItemUsecase(t, cartRepo, nil, gen, now).createGuestCart(t.Context(), now)

			require.ErrorIs(t, err, cart.ErrInvalidSessionToken)
		})
	})
}

func Test_markCreationRace(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("衝突にはやり直しの印を付ける", func(t *testing.T) {
			t.Parallel()

			actual := markCreationRace(xerrors.Wrap(apperror.ErrConflict, "duplicate key"))

			require.ErrorIs(t, actual, errCartCreationRace)
			// 元の衝突も残す。やり直しが尽きたときに 409 として返るのはこちら。
			require.ErrorIs(t, actual, apperror.ErrConflict)
		})

		t.Run("衝突でない失敗には印を付けない", func(t *testing.T) {
			t.Parallel()

			actual := markCreationRace(xerrors.Wrap(apperror.ErrInternal, "connection reset"))

			require.NotErrorIs(t, actual, errCartCreationRace)
			require.ErrorIs(t, actual, apperror.ErrInternal)
		})
	})
}
