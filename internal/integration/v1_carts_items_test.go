package integration

import (
	"context"
	"io"
	"net/http"
	"testing"

	items "go-boilerplate/internal/controller/handler/v1/carts/items"
	"go-boilerplate/internal/controller/handler/v1/carts/items/gen"
	"go-boilerplate/internal/observability"
	cartuc "go-boilerplate/internal/usecase/cart"
	mock_cartuc "go-boilerplate/internal/usecase/cart/mock"
	"go-boilerplate/pkg/ptr"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// meCartItemPath は、明細を設定するエンドポイントのパス。
const meCartItemPath = "/v1/carts/me/items/0198a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b"

func TestV1CartsMeItems_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証済みで明細を設定するとCartResponseが返る", func(t *testing.T) {
			t.Parallel()

			e := echoWithItemsHandler(t, func(uc *mock_cartuc.MockUsecase) {
				uc.EXPECT().SetItem(gomock.Any(), gomock.Any()).Return(cartuc.CartView{
					Items:          []cartuc.CartItemView{},
					SubtotalAmount: 3998,
				}, nil)
			})

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_ci_user"))
			actual := StartServer(t, e).DoJSON(http.MethodPut, meCartItemPath, putBody(2), headers)

			assert.Equal(t, http.StatusOK, actual.StatusCode)
			AssertJSONResponseType[gen.CartResponse](t, actual)
		})

		t.Run("未認証でもX-Cart-Sessionヘッダの主体で200を返す", func(t *testing.T) {
			t.Parallel()

			var captured cartuc.SetItemParams
			e := echoWithItemsHandler(t, func(uc *mock_cartuc.MockUsecase) {
				uc.EXPECT().SetItem(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, params cartuc.SetItemParams) (cartuc.CartView, error) {
						captured = params
						return cartuc.CartView{Items: []cartuc.CartItemView{}}, nil
					},
				)
			})

			headers := http.Header{cartSessionHeader: []string{guestSessionToken}}
			actual := StartServer(t, e).DoJSON(http.MethodPut, meCartItemPath, putBody(2), headers)

			assert.Equal(t, http.StatusOK, actual.StatusCode)
			assert.Nil(t, captured.Subject.UserID)
			require.NotNil(t, captured.Subject.SessionToken)
			assert.Equal(t, guestSessionToken, *captured.Subject.SessionToken)
			assert.Equal(t, 2, captured.Quantity)
		})

		t.Run("認証済みで明細を削除すると204が返り本文を持たない", func(t *testing.T) {
			t.Parallel()

			var captured cartuc.RemoveItemParams
			e := echoWithItemsHandler(t, func(uc *mock_cartuc.MockUsecase) {
				uc.EXPECT().RemoveItem(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, params cartuc.RemoveItemParams) error {
						captured = params
						return nil
					},
				)
			})

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_cd_user"))
			actual := StartServer(t, e).DoJSON(http.MethodDelete, meCartItemPath, nil, headers)

			assert.Equal(t, http.StatusNoContent, actual.StatusCode)
			assert.Empty(t, readBody(t, actual))
			assert.NotNil(t, captured.Subject.UserID)
		})

		t.Run("未認証でもX-Cart-Sessionヘッダの主体で204を返す", func(t *testing.T) {
			t.Parallel()

			var captured cartuc.RemoveItemParams
			e := echoWithItemsHandler(t, func(uc *mock_cartuc.MockUsecase) {
				uc.EXPECT().RemoveItem(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, params cartuc.RemoveItemParams) error {
						captured = params
						return nil
					},
				)
			})

			headers := http.Header{cartSessionHeader: []string{guestSessionToken}}
			actual := StartServer(t, e).DoJSON(http.MethodDelete, meCartItemPath, nil, headers)

			assert.Equal(t, http.StatusNoContent, actual.StatusCode)
			assert.Nil(t, captured.Subject.UserID)
			require.NotNil(t, captured.Subject.SessionToken)
			assert.Equal(t, guestSessionToken, *captured.Subject.SessionToken)
		})

		t.Run("主体を持たない削除も204を返しトークンを発行しない", func(t *testing.T) {
			t.Parallel()

			e := echoWithItemsHandler(t, func(uc *mock_cartuc.MockUsecase) {
				uc.EXPECT().RemoveItem(gomock.Any(), gomock.Any()).Return(nil)
			})

			actual := StartServer(t, e).DoJSON(http.MethodDelete, meCartItemPath, nil, nil)

			assert.Equal(t, http.StatusNoContent, actual.StatusCode)
			assert.Empty(t, readBody(t, actual))
		})

		t.Run("主体を持たない呼び出しには発行されたトークンが返る", func(t *testing.T) {
			t.Parallel()

			e := echoWithItemsHandler(t, func(uc *mock_cartuc.MockUsecase) {
				uc.EXPECT().SetItem(gomock.Any(), gomock.Any()).Return(cartuc.CartView{
					SessionToken: ptr.To(guestSessionToken),
					Items:        []cartuc.CartItemView{},
				}, nil)
			})

			actual := StartServer(t, e).DoJSON(http.MethodPut, meCartItemPath, putBody(1), nil)

			assert.Equal(t, http.StatusOK, actual.StatusCode)
			AssertJSONResponseType[gen.CartResponse](t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("数量が範囲外なら400で弾かれUsecaseは呼ばれない", func(t *testing.T) {
			t.Parallel()

			// spec は業務規則違反を 422 と定めるが、数量の範囲は OpenAPI が宣言しているため
			// ドメインへ届く前に 400 で落ちる（ADR-0016・境界の権威は spec）。
			t.Run("0は削除ではなく範囲外", func(t *testing.T) {
				t.Parallel()
				assertItemsRequestRejected(t, putBody(0))
			})

			t.Run("上限を超える数量", func(t *testing.T) {
				t.Parallel()
				assertItemsRequestRejected(t, putBody(100))
			})

			t.Run("負の数量", func(t *testing.T) {
				t.Parallel()
				assertItemsRequestRejected(t, putBody(-1))
			})
		})

		t.Run("数量が無い本文は400で弾かれる", func(t *testing.T) {
			t.Parallel()

			assertItemsRequestRejected(t, map[string]any{})
		})

		t.Run("宣言に無いフィールドを含む本文は400で弾かれる", func(t *testing.T) {
			t.Parallel()

			assertItemsRequestRejected(t, map[string]any{"quantity": 1, "productId": "spoofed"})
		})

		t.Run("カートへ入れられない商品は422を返す", func(t *testing.T) {
			t.Parallel()

			e := echoWithItemsHandler(t, func(uc *mock_cartuc.MockUsecase) {
				uc.EXPECT().SetItem(gomock.Any(), gomock.Any()).
					Return(cartuc.CartView{}, cartuc.ErrUnavailableProduct)
			})
			UseAppErrorHandler(t, e)

			actual := StartServer(t, e).DoJSON(http.MethodPut, meCartItemPath, putBody(1), nil)

			AssertErrorResponse(t, actual, http.StatusUnprocessableEntity)
		})

		t.Run("無効な資格情報は匿名として通さず401を返す", func(t *testing.T) {
			t.Parallel()

			e := echoWithItemsHandler(t, func(uc *mock_cartuc.MockUsecase) {
				uc.EXPECT().SetItem(gomock.Any(), gomock.Any()).Times(0)
			})
			UseAppErrorHandler(t, e)
			useOpenAPIValidationWithAuthFailure(t, e)

			headers := http.Header{"Authorization": []string{"Bearer invalid-token"}}
			actual := StartServer(t, e).DoJSON(http.MethodPut, meCartItemPath, putBody(1), headers)

			AssertErrorResponse(t, actual, http.StatusUnauthorized)
		})

		t.Run("削除で無効な資格情報は匿名として通さず401を返す", func(t *testing.T) {
			t.Parallel()

			e := echoWithItemsHandler(t, func(uc *mock_cartuc.MockUsecase) {
				uc.EXPECT().RemoveItem(gomock.Any(), gomock.Any()).Times(0)
			})
			UseAppErrorHandler(t, e)
			useOpenAPIValidationWithAuthFailure(t, e)

			headers := http.Header{"Authorization": []string{"Bearer invalid-token"}}
			actual := StartServer(t, e).DoJSON(http.MethodDelete, meCartItemPath, nil, headers)

			AssertErrorResponse(t, actual, http.StatusUnauthorized)
		})

		t.Run("GETは405を返しUsecaseは呼ばれない", func(t *testing.T) {
			t.Parallel()

			e := echoWithItemsHandler(t, func(uc *mock_cartuc.MockUsecase) {
				uc.EXPECT().SetItem(gomock.Any(), gomock.Any()).Times(0)
			})
			UseAppErrorHandler(t, e)

			actual := StartServer(t, e).DoJSON(http.MethodGet, meCartItemPath, nil, nil)

			AssertErrorResponse(t, actual, http.StatusMethodNotAllowed)
		})
	})
}

// readBody は、応答の本文を読み切って文字列で返します。
func readBody(t *testing.T, response *http.Response) string {
	t.Helper()
	defer func() { require.NoError(t, response.Body.Close()) }()

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	return string(body)
}

// putBody は、数量だけを持つ設定リクエストの本文を組み立てます。
func putBody(quantity int) map[string]any {
	return map[string]any{"quantity": quantity}
}

// echoWithItemsHandler は、モックしたユースケースを結線した Echo を返します。
func echoWithItemsHandler(t *testing.T, expect func(*mock_cartuc.MockUsecase)) *echo.Echo {
	t.Helper()

	e := echo.New()
	uc := mock_cartuc.NewMockUsecase(gomock.NewController(t))
	expect(uc)
	items.BindHandler(e, observability.NewNoopTracerFactory(t), uc)
	return e
}

// assertItemsRequestRejected は、実 OpenAPI ミドルウェア経由で本文が 400 で拒否され、
// ユースケースまで到達しないことを検証します。
func assertItemsRequestRejected(t *testing.T, body any) {
	t.Helper()

	e := echoWithItemsHandler(t, func(uc *mock_cartuc.MockUsecase) {
		uc.EXPECT().SetItem(gomock.Any(), gomock.Any()).Times(0)
	})
	UseAppErrorHandler(t, e)
	useOpenAPIValidation(t, e)

	actual := StartServer(t, e).DoJSON(http.MethodPut, meCartItemPath, body, nil)

	AssertErrorResponse(t, actual, http.StatusBadRequest)
}
