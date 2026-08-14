package carts

import (
	"context"
	"math"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	"go-boilerplate/internal/controller/handler/v1/carts/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	cartuc "go-boilerplate/internal/usecase/cart"
	mock_cartuc "go-boilerplate/internal/usecase/cart/mock"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// testSessionToken は、ヘッダで受け取るゲストセッショントークンのサンプル値。
const testSessionToken = "abcdefghijklmnopqrstuvwxyz0123456789-_ABCDE"

// authnContext は、内部ユーザー ID を解決済みの認証コンテキストを返すテストヘルパーです。
func authnContext(t *testing.T, userID uuid.UUID) context.Context {
	t.Helper()
	ctx := ctxhelper.WithAuthn(context.Background())
	authn, err := auth.New("subject", "issuer", nil, nil)
	require.NoError(t, err)

	resolved, err := authn.WithUserID(userID)
	require.NoError(t, err)
	require.True(t, ctxhelper.SetAuthn(ctx, *resolved))
	return ctx
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)
	uc := mock_cartuc.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, tf, uc)

	routes := e.Router().Routes()
	testassert.AssertEchoRouterPath(t, "/v1/carts/me", routes)
	testassert.AssertEchoRouterMethods(t, []string{http.MethodGet, http.MethodDelete}, routes)
}

func Test_server_GetCartsMe(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証主体のuserIDをユースケースへ渡しカートを200で返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_cartuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			userID := uuidtestkit.NewTestFromSalt(t, "hc_user")
			uc.EXPECT().GetCart(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, subject cartuc.Subject) (cartuc.CartView, error) {
					require.NotNil(t, subject.UserID)
					assert.Equal(t, userID, *subject.UserID)
					assert.Nil(t, subject.SessionToken)
					return cartuc.CartView{Items: []cartuc.CartItemView{}, SubtotalAmount: 1200}, nil
				})

			resp, err := s.GetCartsMe(authnContext(t, userID), gen.GetCartsMeRequestObject{})
			require.NoError(t, err)

			r, ok := resp.(gen.GetCartsMe200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, int64(1200), r.SubtotalAmount)
			assert.Empty(t, r.Items)
		})

		t.Run("未認証はヘッダのセッショントークンを主体として渡す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_cartuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			uc.EXPECT().GetCart(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, subject cartuc.Subject) (cartuc.CartView, error) {
					assert.Nil(t, subject.UserID)
					require.NotNil(t, subject.SessionToken)
					assert.Equal(t, testSessionToken, *subject.SessionToken)
					return cartuc.CartView{Items: []cartuc.CartItemView{}}, nil
				})

			_, err := s.GetCartsMe(context.Background(), gen.GetCartsMeRequestObject{
				Params: gen.GetCartsMeParams{XCartSession: ptr.To(testSessionToken)},
			})
			require.NoError(t, err)
		})

		t.Run("認証済みならヘッダのセッショントークンを無視する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_cartuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			userID := uuidtestkit.NewTestFromSalt(t, "hc_user_both")
			uc.EXPECT().GetCart(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, subject cartuc.Subject) (cartuc.CartView, error) {
					// 両方を詰めると、どちらのカートを指すのかがユースケース側で決められなくなる。
					require.NotNil(t, subject.UserID)
					assert.Nil(t, subject.SessionToken)
					return cartuc.CartView{Items: []cartuc.CartItemView{}}, nil
				})

			_, err := s.GetCartsMe(authnContext(t, userID), gen.GetCartsMeRequestObject{
				Params: gen.GetCartsMeParams{XCartSession: ptr.To(testSessionToken)},
			})
			require.NoError(t, err)
		})

		t.Run("認証もヘッダも無い場合は主体を持たないまま呼ぶ", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_cartuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			uc.EXPECT().GetCart(gomock.Any(), cartuc.Subject{}).
				Return(cartuc.CartView{Items: []cartuc.CartItemView{}}, nil)

			_, err := s.GetCartsMe(context.Background(), gen.GetCartsMeRequestObject{})
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証済みだが内部UserIDが未解決の場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_cartuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetCart(gomock.Any(), gomock.Any()).Times(0)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			ctx := ctxhelper.WithAuthn(context.Background())
			authn, err := auth.New("subject", "issuer", nil, nil)
			require.NoError(t, err)
			require.True(t, ctxhelper.SetAuthn(ctx, *authn))

			_, err = s.GetCartsMe(ctx, gen.GetCartsMeRequestObject{})

			require.ErrorIs(t, err, auth.ErrUserIDUnresolved)
		})

		t.Run("ユースケースがエラーを返した場合はそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_cartuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			uc.EXPECT().GetCart(gomock.Any(), gomock.Any()).Return(cartuc.CartView{}, apperror.ErrInternal)

			_, err := s.GetCartsMe(context.Background(), gen.GetCartsMeRequestObject{})

			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("明細の数量がint32へ収まらない場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_cartuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			uc.EXPECT().GetCart(gomock.Any(), gomock.Any()).Return(cartuc.CartView{
				Items: []cartuc.CartItemView{{Quantity: math.MaxInt32 + 1}},
			}, nil)

			_, err := s.GetCartsMe(context.Background(), gen.GetCartsMeRequestObject{})

			require.ErrorIs(t, err, safecast.ErrOverflow)
		})
	})
}

func Test_server_DeleteCartsMe(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証主体のuserIDをユースケースへ渡し204で返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_cartuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			userID := uuidtestkit.NewTestFromSalt(t, "hc_clear_user")
			uc.EXPECT().ClearCart(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, subject cartuc.Subject) error {
					require.NotNil(t, subject.UserID)
					assert.Equal(t, userID, *subject.UserID)
					assert.Nil(t, subject.SessionToken)
					return nil
				})

			resp, err := s.DeleteCartsMe(authnContext(t, userID), gen.DeleteCartsMeRequestObject{})
			require.NoError(t, err)

			assert.IsType(t, gen.DeleteCartsMe204Response{}, resp)
		})

		t.Run("未認証はヘッダのセッショントークンを主体として渡す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_cartuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			uc.EXPECT().ClearCart(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, subject cartuc.Subject) error {
					assert.Nil(t, subject.UserID)
					require.NotNil(t, subject.SessionToken)
					assert.Equal(t, testSessionToken, *subject.SessionToken)
					return nil
				})

			resp, err := s.DeleteCartsMe(context.Background(), gen.DeleteCartsMeRequestObject{
				Params: gen.DeleteCartsMeParams{XCartSession: ptr.To(testSessionToken)},
			})
			require.NoError(t, err)

			assert.IsType(t, gen.DeleteCartsMe204Response{}, resp)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証済みだが内部UserIDが未解決の場合はユースケースを呼ばない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_cartuc.NewMockUsecase(ctrl)
			uc.EXPECT().ClearCart(gomock.Any(), gomock.Any()).Times(0)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			ctx := ctxhelper.WithAuthn(context.Background())
			authn, err := auth.New("subject", "issuer", nil, nil)
			require.NoError(t, err)
			require.True(t, ctxhelper.SetAuthn(ctx, *authn))

			_, err = s.DeleteCartsMe(ctx, gen.DeleteCartsMeRequestObject{})

			require.ErrorIs(t, err, auth.ErrUserIDUnresolved)
		})

		t.Run("ユースケースがエラーを返した場合はそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_cartuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			uc.EXPECT().ClearCart(gomock.Any(), gomock.Any()).Return(apperror.ErrInternal)

			_, err := s.DeleteCartsMe(context.Background(), gen.DeleteCartsMeRequestObject{})

			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}

func Test_toSubject(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証コンテキストがあればuserIDを主体にする", func(t *testing.T) {
			t.Parallel()

			userID := uuidtestkit.NewTestFromSalt(t, "sub_user")

			actual, err := toSubject(authnContext(t, userID), nil)
			require.NoError(t, err)

			require.NotNil(t, actual.UserID)
			assert.Equal(t, userID, *actual.UserID)
			assert.Nil(t, actual.SessionToken)
		})

		t.Run("認証コンテキストが無ければヘッダを主体にする", func(t *testing.T) {
			t.Parallel()

			actual, err := toSubject(context.Background(), ptr.To(testSessionToken))
			require.NoError(t, err)

			assert.Nil(t, actual.UserID)
			require.NotNil(t, actual.SessionToken)
			assert.Equal(t, testSessionToken, *actual.SessionToken)
		})

		t.Run("認証もヘッダも無ければ主体を持たない", func(t *testing.T) {
			t.Parallel()

			actual, err := toSubject(context.Background(), nil)
			require.NoError(t, err)

			assert.Equal(t, cartuc.Subject{}, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("内部UserIDが未解決の認証コンテキストではエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctx := ctxhelper.WithAuthn(context.Background())
			authn, err := auth.New("subject", "issuer", nil, nil)
			require.NoError(t, err)
			require.True(t, ctxhelper.SetAuthn(ctx, *authn))

			_, err = toSubject(ctx, nil)

			require.ErrorIs(t, err, auth.ErrUserIDUnresolved)
		})
	})
}

func Test_toCartResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ビューの値をレスポンスへ写す", func(t *testing.T) {
			t.Parallel()

			expiresAt := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)

			actual, err := toCartResponse(cartuc.CartView{
				SessionToken:   ptr.To(testSessionToken),
				Items:          []cartuc.CartItemView{},
				SubtotalAmount: 3998,
				ExpiresAt:      &expiresAt,
			})
			require.NoError(t, err)

			require.NotNil(t, actual.SessionToken)
			assert.Equal(t, testSessionToken, *actual.SessionToken)
			assert.Equal(t, int64(3998), actual.SubtotalAmount)
			require.NotNil(t, actual.ExpiresAt)
			assert.Equal(t, expiresAt, *actual.ExpiresAt)
			assert.Empty(t, actual.Items)
		})

		t.Run("カートが無い場合はセッショントークンと有効期限がnullになる", func(t *testing.T) {
			t.Parallel()

			actual, err := toCartResponse(cartuc.CartView{Items: []cartuc.CartItemView{}})
			require.NoError(t, err)

			assert.Nil(t, actual.SessionToken)
			assert.Nil(t, actual.ExpiresAt)
			assert.Zero(t, actual.SubtotalAmount)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("明細の変換に失敗した場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			_, err := toCartResponse(cartuc.CartView{
				Items: []cartuc.CartItemView{{Quantity: math.MaxInt32 + 1}},
			})

			require.ErrorIs(t, err, safecast.ErrOverflow)
		})
	})
}

func Test_toCartItemResponses(t *testing.T) {
	t.Parallel()

	productID := uuidtestkit.NewTestFromSalt(t, "item_product")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("明細の値をレスポンスへ写す", func(t *testing.T) {
			t.Parallel()

			price, err := decimal.Parse("19.99")
			require.NoError(t, err)

			actual, err := toCartItemResponses([]cartuc.CartItemView{{
				ProductID:         productID,
				ProductName:       ptr.To("エルゴノミクスチェア"),
				Quantity:          2,
				UnitPrice:         &price,
				Issues:            []cartuc.ItemIssue{cartuc.ItemIssueInsufficientStock},
				AvailableQuantity: ptr.To(1),
			}})
			require.NoError(t, err)

			require.Len(t, actual, 1)
			assert.Equal(t, productID.ToPrimitive(), actual[0].ProductId)
			require.NotNil(t, actual[0].ProductName)
			assert.Equal(t, "エルゴノミクスチェア", *actual[0].ProductName)
			assert.Equal(t, int32(2), actual[0].Quantity)
			require.NotNil(t, actual[0].UnitPrice)
			assert.Equal(t, "19.99", *actual[0].UnitPrice)
			assert.Equal(t, []gen.CartItemIssue{gen.InsufficientStock}, actual[0].Issues)
			require.NotNil(t, actual[0].AvailableQuantity)
			assert.Equal(t, int32(1), *actual[0].AvailableQuantity)
		})

		t.Run("商品を引けなかった明細は商品名と単価がnullになる", func(t *testing.T) {
			t.Parallel()

			actual, err := toCartItemResponses([]cartuc.CartItemView{{
				ProductID: productID,
				Quantity:  1,
				Issues:    []cartuc.ItemIssue{cartuc.ItemIssueNotFound},
			}})
			require.NoError(t, err)

			require.Len(t, actual, 1)
			assert.Nil(t, actual[0].ProductName)
			assert.Nil(t, actual[0].UnitPrice)
			assert.Nil(t, actual[0].AvailableQuantity)
		})

		t.Run("明細が無い場合はnilではない空スライスを返す", func(t *testing.T) {
			t.Parallel()

			actual, err := toCartItemResponses(nil)
			require.NoError(t, err)

			assert.NotNil(t, actual)
			assert.Empty(t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("数量がint32へ収まらない場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			_, err := toCartItemResponses([]cartuc.CartItemView{{Quantity: math.MaxInt32 + 1}})

			require.ErrorIs(t, err, safecast.ErrOverflow)
		})

		t.Run("購入可能上限がint32へ収まらない場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			_, err := toCartItemResponses([]cartuc.CartItemView{{
				Quantity:          1,
				AvailableQuantity: ptr.To(math.MaxInt32 + 1),
			}})

			require.ErrorIs(t, err, safecast.ErrOverflow)
		})
	})
}

func Test_toAvailableQuantity(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定された上限をint32へ変換する", func(t *testing.T) {
			t.Parallel()

			actual, err := toAvailableQuantity(ptr.To(3))
			require.NoError(t, err)

			require.NotNil(t, actual)
			assert.Equal(t, int32(3), *actual)
		})

		t.Run("指定が無い場合はnilを返す", func(t *testing.T) {
			t.Parallel()

			actual, err := toAvailableQuantity(nil)
			require.NoError(t, err)

			assert.Nil(t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("int32へ収まらない場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			_, err := toAvailableQuantity(ptr.To(math.MaxInt32 + 1))

			require.ErrorIs(t, err, safecast.ErrOverflow)
		})
	})
}

func Test_toUnitPrice(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("単価をdecimal文字列へ変換する", func(t *testing.T) {
			t.Parallel()

			price, err := decimal.Parse("19.99")
			require.NoError(t, err)

			actual := toUnitPrice(&price)

			require.NotNil(t, actual)
			assert.Equal(t, "19.99", *actual)
		})

		t.Run("単価が無い場合はnilを返す", func(t *testing.T) {
			t.Parallel()

			assert.Nil(t, toUnitPrice(nil))
		})
	})
}

func Test_toCartItemIssues(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ユースケースの値をレスポンスの列挙値へ写す", func(t *testing.T) {
			t.Parallel()

			actual := toCartItemIssues([]cartuc.ItemIssue{
				cartuc.ItemIssueNotFound,
				cartuc.ItemIssueUnpublished,
				cartuc.ItemIssueOutOfStock,
				cartuc.ItemIssueInsufficientStock,
				cartuc.ItemIssuePriceIncreased,
				cartuc.ItemIssuePriceDecreased,
			})

			// 両層の値が一致していることが、この写像がキャストで済む前提そのもの。
			assert.Equal(t, []gen.CartItemIssue{
				gen.NotFound,
				gen.Unpublished,
				gen.OutOfStock,
				gen.InsufficientStock,
				gen.PriceIncreased,
				gen.PriceDecreased,
			}, actual)
		})

		t.Run("issueが無い場合はnilではない空スライスを返す", func(t *testing.T) {
			t.Parallel()

			actual := toCartItemIssues(nil)

			assert.NotNil(t, actual)
			assert.Empty(t, actual)
		})
	})
}
