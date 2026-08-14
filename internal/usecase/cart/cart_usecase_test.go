package cart

import (
	"strings"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/cart"
	mock_cart "go-boilerplate/internal/domain/cart/mock"
	"go-boilerplate/internal/domain/lexicon/money"
	"go-boilerplate/internal/domain/product"
	mock_product "go-boilerplate/internal/domain/product/mock"
	"go-boilerplate/internal/observability"
	clocktestkit "go-boilerplate/internal/usecase/boundary/clock/testkit"
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

// testSessionToken は、形式検証を通る 43 文字の base64url トークン。
const testSessionToken = "abcdefghijklmnopqrstuvwxyz0123456789-_ABCDE"

func newPrice(t *testing.T, amount string) money.Price {
	t.Helper()
	d, err := decimal.Parse(amount)
	require.NoError(t, err)
	p, err := money.NewPrice(d)
	require.NoError(t, err)
	return p
}

// newTestProduct は、公開中の商品を組み立てるテストヘルパーです。
// 公開状態による判定はドメインが持つため、この層のテストで非公開を作り分ける必要はありません。
func newTestProduct(t *testing.T, id uuid.UUID, amount string, quantity int) *product.Product {
	t.Helper()

	status, err := product.NewStatusRef(uuidtestkit.NewTestFromSalt(t, "cart_status"), "販売中")
	require.NoError(t, err)
	category, err := product.NewCategoryRef(uuidtestkit.NewTestFromSalt(t, "cart_category"), "家具")
	require.NoError(t, err)

	p, err := product.New(id, product.Attributes{
		Name:        "テスト商品",
		Price:       newPrice(t, amount),
		Quantity:    quantity,
		Status:      status,
		Category:    category,
		PublishedAt: ptr.To(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)
	return p
}

func newTestCartItem(t *testing.T, salt string, productID uuid.UUID, quantity int, lastSeen *money.Price) cart.CartItem {
	t.Helper()
	return cart.NewCartItem(uuidtestkit.NewTestFromSalt(t, salt), cart.CartItemAttributes{
		ProductID:     productID,
		Quantity:      quantity,
		AddedAt:       time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		LastSeenPrice: lastSeen,
	})
}

func newGuestCart(t *testing.T, expiresAt time.Time, items ...cart.CartItem) *cart.Cart {
	t.Helper()

	token, err := cart.NewSessionToken(testSessionToken)
	require.NoError(t, err)

	c, err := cart.Reconstruct(uuidtestkit.NewTestFromSalt(t, "cart_guest"), cart.Attributes{
		SessionToken: &token,
		Items:        items,
		ExpiresAt:    expiresAt,
	})
	require.NoError(t, err)
	return c
}

func newTestUsecase(t *testing.T, cartRepo cart.Repository, productRepo product.Repository) *usecase {
	t.Helper()
	return &usecase{
		tracer:      observability.NewNoopTracerFactory(t).Usecase(),
		cartRepo:    cartRepo,
		productRepo: productRepo,
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("受け取った依存をそのまま保持した実装を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			txm := mock_tx.NewMockManager(ctrl)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			productRepo := mock_product.NewMockRepository(ctrl)
			clk := clocktestkit.NewMockClock(t, time.Time{})

			actual, ok := New(txm, cartRepo, productRepo, clk, observability.NewNoopTracerFactory(t)).(*usecase)
			require.True(t, ok)

			assert.Equal(t, txm, actual.txm)
			assert.Equal(t, cartRepo, actual.cartRepo)
			assert.Equal(t, productRepo, actual.productRepo)
			assert.Equal(t, clk, actual.clock)
			assert.NotNil(t, actual.tracer)
		})
	})
}

func Test_emptyCartView(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("明細は空スライスで有効期限を持たない", func(t *testing.T) {
			t.Parallel()

			actual := emptyCartView()

			// nil ではなく空スライスであることが、JSON で null ではなく [] になる条件。
			assert.NotNil(t, actual.Items)
			assert.Empty(t, actual.Items)
			assert.Nil(t, actual.ExpiresAt)
			assert.Nil(t, actual.SessionToken)
			assert.Zero(t, actual.SubtotalAmount)
		})
	})
}

func Test_evaluateItem(t *testing.T) {
	t.Parallel()

	productID := uuidtestkit.NewTestFromSalt(t, "cart_eval_product")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("商品の観測値をドメインへ渡し結果を DTO へ写す", func(t *testing.T) {
			t.Parallel()

			seen := newPrice(t, "10.00")
			item := newTestCartItem(t, "cart_eval_delegate", productID, 3, &seen)

			// 在庫不足かつ値上がりになる観測値を渡し、判定結果がそのまま写っていることを確かめる。
			actual := evaluateItem(item, newTestProduct(t, productID, "12.00", 1))

			assert.Equal(t, []ItemIssue{ItemIssueInsufficientStock, ItemIssuePriceIncreased}, actual.Issues)
			require.NotNil(t, actual.AvailableQuantity)
			assert.Equal(t, 1, *actual.AvailableQuantity)
			assert.Equal(t, productID, actual.ProductID)
			assert.Equal(t, 3, actual.Quantity)
			require.NotNil(t, actual.ProductName)
			assert.Equal(t, "テスト商品", *actual.ProductName)
			require.NotNil(t, actual.UnitPrice)
			assert.Equal(t, "12", actual.UnitPrice.String())
		})

		t.Run("商品を引けない場合は商品名と単価を持たない", func(t *testing.T) {
			t.Parallel()

			item := newTestCartItem(t, "cart_eval_notfound", productID, 2, nil)

			actual := evaluateItem(item, nil)

			assert.Equal(t, []ItemIssue{ItemIssueNotFound}, actual.Issues)
			assert.Nil(t, actual.ProductName)
			assert.Nil(t, actual.UnitPrice)
			assert.Nil(t, actual.AvailableQuantity)
		})

		t.Run("問題が無ければ issue は空スライスになる", func(t *testing.T) {
			t.Parallel()

			item := newTestCartItem(t, "cart_eval_ok", productID, 1, nil)

			actual := evaluateItem(item, newTestProduct(t, productID, "10.00", 5))

			// nil ではなく空スライスであることが、JSON で null ではなく [] になる条件。
			assert.NotNil(t, actual.Issues)
			assert.Empty(t, actual.Issues)
		})
	})
}

func Test_toItemIssues(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ドメインの値を DTO の語彙へ写す", func(t *testing.T) {
			t.Parallel()

			actual := toItemIssues([]cart.Issue{cart.IssueOutOfStock, cart.IssuePriceDecreased})

			assert.Equal(t, []ItemIssue{ItemIssueOutOfStock, ItemIssuePriceDecreased}, actual)
		})

		t.Run("空の場合は nil ではない空スライスを返す", func(t *testing.T) {
			t.Parallel()

			actual := toItemIssues(nil)

			assert.NotNil(t, actual)
			assert.Empty(t, actual)
		})
	})
}

func Test_toItemIssue(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("すべての値が DTO の語彙へ対応する", func(t *testing.T) {
			t.Parallel()

			// 対応が 1 つでも欠けると panic するため、全値を通すことが写像の網羅性の担保になる。
			assert.Equal(t, ItemIssueNotFound, toItemIssue(cart.IssueNotFound))
			assert.Equal(t, ItemIssueUnpublished, toItemIssue(cart.IssueUnpublished))
			assert.Equal(t, ItemIssueOutOfStock, toItemIssue(cart.IssueOutOfStock))
			assert.Equal(t, ItemIssueInsufficientStock, toItemIssue(cart.IssueInsufficientStock))
			assert.Equal(t, ItemIssuePriceIncreased, toItemIssue(cart.IssuePriceIncreased))
			assert.Equal(t, ItemIssuePriceDecreased, toItemIssue(cart.IssuePriceDecreased))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("対応を持たない値は panic する", func(t *testing.T) {
			t.Parallel()

			// 黙って既定値へ倒すと、ドメインに値が増えたとき応答だけが静かに嘘になる。
			assert.Panics(t, func() { _ = toItemIssue(cart.Issue("unknown")) })
		})
	})
}

func Test_evaluateItems(t *testing.T) {
	t.Parallel()

	foundID := uuidtestkit.NewTestFromSalt(t, "cart_items_found")
	missingID := uuidtestkit.NewTestFromSalt(t, "cart_items_missing")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("引けた商品の価格だけを提示価格の表へ入れる", func(t *testing.T) {
			t.Parallel()

			items := []cart.CartItem{
				newTestCartItem(t, "cart_items_a", foundID, 1, nil),
				newTestCartItem(t, "cart_items_b", missingID, 1, nil),
			}
			products := map[uuid.UUID]*product.Product{
				foundID: newTestProduct(t, foundID, "10.00", 5),
			}

			views, seen := evaluateItems(items, products)

			assert.Len(t, views, 2)
			// 引けなかった明細を表へ入れると、その明細の提示価格が MarkSeen で消される。
			assert.Len(t, seen, 1)
			assert.Contains(t, seen, foundID)
			assert.NotContains(t, seen, missingID)
		})

		t.Run("明細が無い場合は空のViewと空の表を返す", func(t *testing.T) {
			t.Parallel()

			views, seen := evaluateItems(nil, nil)

			assert.NotNil(t, views)
			assert.Empty(t, views)
			assert.Empty(t, seen)
		})
	})
}

func Test_subtotalAmount(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購入可能な明細のみを合算する", func(t *testing.T) {
			t.Parallel()

			views := []CartItemView{
				{UnitPrice: ptr.To(decimal.FromInt(10)), Quantity: 2},
				{UnitPrice: ptr.To(decimal.FromInt(50)), Quantity: 1, Issues: []ItemIssue{ItemIssueOutOfStock}},
			}

			actual, err := subtotalAmount(views)
			require.NoError(t, err)

			assert.Equal(t, int64(2000), actual)
		})

		t.Run("合算してから丸めるため明細ごとの丸め誤差が積み上がらない", func(t *testing.T) {
			t.Parallel()

			price, err := decimal.Parse("0.005")
			require.NoError(t, err)
			views := []CartItemView{
				{UnitPrice: &price, Quantity: 1},
				{UnitPrice: &price, Quantity: 1},
			}

			actual, err := subtotalAmount(views)
			require.NoError(t, err)

			// 明細ごとに丸めると 1 + 1 = 2 セントになる。合算してから丸めれば 0.01 ドル = 1 セント。
			assert.Equal(t, int64(1), actual)
		})

		t.Run("合算対象が無ければ0を返す", func(t *testing.T) {
			t.Parallel()

			actual, err := subtotalAmount(nil)
			require.NoError(t, err)

			assert.Zero(t, actual)
		})

		t.Run("単価を持たない明細は合算しない", func(t *testing.T) {
			t.Parallel()

			// issues が空でも単価が無ければ掛ける値が無い。nil 参照を避ける防御が効いていることを固定する。
			views := []CartItemView{
				{UnitPrice: nil, Quantity: 3},
				{UnitPrice: ptr.To(decimal.FromInt(7)), Quantity: 1},
			}

			actual, err := subtotalAmount(views)
			require.NoError(t, err)

			assert.Equal(t, int64(700), actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("決済スケールへ落とせない大きさなら分類済みエラーを返す", func(t *testing.T) {
			t.Parallel()

			// 単価 1 件は money.Price の構築時に検証されるため、幅を超えるのは合算の結果に限られる。
			// 上限ちょうどの単価を 2 明細ぶん積み、黙って壊れた値を返さないことを固定する。
			price, err := decimal.Parse("92233720368547758.07")
			require.NoError(t, err)

			_, err = subtotalAmount([]CartItemView{
				{UnitPrice: &price, Quantity: 1},
				{UnitPrice: &price, Quantity: 1},
			})

			// 未分類のまま外へ出ると 500 の理由が追えないため、apperror へ分類されていることを固定する。
			require.ErrorIs(t, err, apperror.ErrInternal)
			require.ErrorIs(t, err, decimal.ErrOverflow)
		})
	})
}

func Test_usecase_findCart(t *testing.T) {
	t.Parallel()

	userID := uuidtestkit.NewTestFromSalt(t, "cart_find_user")
	expiresAt := time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証済み主体は所有者でカートを引く", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			expected := newGuestCart(t, expiresAt)

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(expected, nil)

			actual, found, err := newTestUsecase(t, cartRepo, nil).findCart(t.Context(), Subject{UserID: &userID})
			require.NoError(t, err)

			assert.True(t, found)
			assert.Same(t, expected, actual)
		})

		t.Run("ゲスト主体はセッショントークンでカートを引く", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			expected := newGuestCart(t, expiresAt)
			token, err := cart.NewSessionToken(testSessionToken)
			require.NoError(t, err)

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), token).Return(expected, nil)

			actual, found, err := newTestUsecase(t, cartRepo, nil).
				findCart(t.Context(), Subject{SessionToken: ptr.To(testSessionToken)})
			require.NoError(t, err)

			assert.True(t, found)
			assert.Same(t, expected, actual)
		})

		t.Run("認証済み主体はゲストトークンより優先される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			expected := newGuestCart(t, expiresAt)

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(expected, nil)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Times(0)

			actual, found, err := newTestUsecase(t, cartRepo, nil).findCart(t.Context(), Subject{
				UserID:       &userID,
				SessionToken: ptr.To(testSessionToken),
			})
			require.NoError(t, err)

			assert.True(t, found)
			assert.Same(t, expected, actual)
		})

		t.Run("主体を持たない場合はカートを引かない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), gomock.Any()).Times(0)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Times(0)

			actual, found, err := newTestUsecase(t, cartRepo, nil).findCart(t.Context(), Subject{})
			require.NoError(t, err)

			assert.False(t, found)
			assert.Nil(t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("セッショントークンの形式が不正なら検証エラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindBySessionToken(gomock.Any(), gomock.Any()).Times(0)

			_, _, err := newTestUsecase(t, cartRepo, nil).
				findCart(t.Context(), Subject{SessionToken: ptr.To(strings.Repeat("a", 10))})

			require.ErrorIs(t, err, cart.ErrInvalidSessionToken)
		})
	})
}

func Test_usecase_findProducts(t *testing.T) {
	t.Parallel()

	productID := uuidtestkit.NewTestFromSalt(t, "cart_findproducts_product")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("明細の商品IDで引き当てた表を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			expected := newTestProduct(t, productID, "10.00", 5)

			productRepo := mock_product.NewMockRepository(ctrl)
			productRepo.EXPECT().FindByIDs(gomock.Any(), []uuid.UUID{productID}).
				Return(product.Products{expected}, nil)

			items := []cart.CartItem{newTestCartItem(t, "cart_findproducts_item", productID, 1, nil)}

			actual, err := newTestUsecase(t, nil, productRepo).findProducts(t.Context(), items)
			require.NoError(t, err)

			assert.Equal(t, map[uuid.UUID]*product.Product{productID: expected}, actual)
		})

		t.Run("明細が無ければ商品を問い合わせない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			productRepo := mock_product.NewMockRepository(ctrl)
			productRepo.EXPECT().FindByIDs(gomock.Any(), gomock.Any()).Times(0)

			actual, err := newTestUsecase(t, nil, productRepo).findProducts(t.Context(), nil)
			require.NoError(t, err)

			assert.Empty(t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("商品の取得に失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			expectedErr := xerrors.New("failed to find products")

			productRepo := mock_product.NewMockRepository(ctrl)
			productRepo.EXPECT().FindByIDs(gomock.Any(), gomock.Any()).Return(nil, expectedErr)

			items := []cart.CartItem{newTestCartItem(t, "cart_findproducts_err", productID, 1, nil)}

			_, err := newTestUsecase(t, nil, productRepo).findProducts(t.Context(), items)

			require.ErrorIs(t, err, expectedErr)
		})
	})
}

func Test_usecase_resolveCart(t *testing.T) {
	t.Parallel()

	userID := uuidtestkit.NewTestFromSalt(t, "cart_resolve_user")
	now := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効なカートをそのまま返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			expected := newGuestCart(t, now.Add(time.Hour))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(expected, nil)

			actual, found, err := newTestUsecase(t, cartRepo, nil).resolveCart(t.Context(), Subject{UserID: &userID}, now)
			require.NoError(t, err)

			assert.True(t, found)
			assert.Same(t, expected, actual)
		})

		t.Run("カートが無ければnilを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).
				Return(nil, xerrors.Wrap(apperror.ErrNotFound, "cart not found"))

			actual, found, err := newTestUsecase(t, cartRepo, nil).resolveCart(t.Context(), Subject{UserID: &userID}, now)
			require.NoError(t, err)

			assert.False(t, found)
			assert.Nil(t, actual)
		})

		t.Run("有効期限を過ぎたカートはnilを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			// SQL では絞らず取得してから判定するため、期限切れの行はここへ届く。
			expired := newGuestCart(t, now.Add(-time.Second))

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(expired, nil)

			actual, found, err := newTestUsecase(t, cartRepo, nil).resolveCart(t.Context(), Subject{UserID: &userID}, now)
			require.NoError(t, err)

			assert.False(t, found)
			assert.Nil(t, actual)
		})

		t.Run("有効期限ちょうどはまだ期限切れではない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			expected := newGuestCart(t, now)

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(expected, nil)

			actual, found, err := newTestUsecase(t, cartRepo, nil).resolveCart(t.Context(), Subject{UserID: &userID}, now)
			require.NoError(t, err)

			assert.True(t, found)
			assert.Same(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("NotFound以外のエラーは伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			expectedErr := xerrors.New("connection refused")

			cartRepo := mock_cart.NewMockRepository(ctrl)
			cartRepo.EXPECT().FindByOwnerID(gomock.Any(), userID).Return(nil, expectedErr)

			_, _, err := newTestUsecase(t, cartRepo, nil).resolveCart(t.Context(), Subject{UserID: &userID}, now)

			require.ErrorIs(t, err, expectedErr)
		})
	})
}
