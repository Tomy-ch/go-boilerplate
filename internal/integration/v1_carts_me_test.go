package integration

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	carts "go-boilerplate/internal/controller/handler/v1/carts"
	"go-boilerplate/internal/controller/handler/v1/carts/gen"
	"go-boilerplate/internal/controller/httpstack/oapi"
	"go-boilerplate/internal/controller/httpstack/oapi/validator"
	"go-boilerplate/internal/observability"
	cartuc "go-boilerplate/internal/usecase/cart"
	mock_cartuc "go-boilerplate/internal/usecase/cart/mock"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	meCartPath = "/v1/carts/me"
	// cartSessionHeader は、ゲストカートの主体を運ぶリクエストヘッダ。
	cartSessionHeader = "X-Cart-Session"
	// guestSessionToken は、ゲストセッショントークンの形式を満たすサンプル値。
	guestSessionToken = "abcdefghijklmnopqrstuvwxyz0123456789-_ABCDE"
)

// useOpenAPIValidationWithAuthFailure は、提示された資格情報の検証が必ず失敗する形で実 OpenAPI
// ミドルウェアを登録します。任意認証の operation が、無効な資格情報を匿名として通さないことを
// 実際のミドルウェア経路で確かめるために使います（ADR-0019 (optional-authentication-fail-closed)）。
func useOpenAPIValidationWithAuthFailure(t *testing.T, e *echo.Echo) {
	t.Helper()

	spec, err := validator.GetValidator()
	require.NoError(t, err)

	skipper := func(*echo.Context) bool { return false }
	authFunc := func(_ context.Context, input *openapi3filter.AuthenticationInput) error {
		// スロットを持つのはリクエストの context だけで、バリデータが渡す context は空
		// （internal/controller/httpstack/oapi/auth/auth.go の NewAuthenticator と同じ理由）。
		req := input.RequestValidationInput.Request
		//nolint:contextcheck // input が内包する request の context のスロットへ書き戻すため
		ctxhelper.SetAuthnFailure(req.Context(), ctxhelper.ErrUnauthenticatedUser)
		return ctxhelper.ErrUnauthenticatedUser
	}
	e.Use(oapi.Middleware(spec, skipper, authFunc))
}

func TestV1CartsMe_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証済みで自分のカートがCartResponseで返る", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_cartuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetCart(gomock.Any(), gomock.Any()).Return(cartuc.CartView{
				Items:          []cartuc.CartItemView{},
				SubtotalAmount: 3998,
			}, nil)

			carts.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_ct_user"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, meCartPath, nil, headers)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			AssertJSONResponseType[gen.CartResponse](t, actual)
		})

		t.Run("取得対象が認証主体のuserIDに限定されている", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			userID := uuidtestkit.NewTestFromSalt(t, "int_ct_owner")
			var captured cartuc.Subject
			uc := mock_cartuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetCart(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, subject cartuc.Subject) (cartuc.CartView, error) {
					captured = subject
					return cartuc.CartView{Items: []cartuc.CartItemView{}}, nil
				},
			)

			carts.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, userID)
			actual := StartServer(t, e).DoJSON(http.MethodGet, meCartPath, nil, headers)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			require.NotNil(t, captured.UserID)
			assert.Equal(t, userID, *captured.UserID)
		})

		t.Run("未認証でもX-Cart-Sessionヘッダの主体で200を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			var captured cartuc.Subject
			uc := mock_cartuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetCart(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, subject cartuc.Subject) (cartuc.CartView, error) {
					captured = subject
					return cartuc.CartView{Items: []cartuc.CartItemView{}}, nil
				},
			)

			carts.BindHandler(e, tf, uc)

			headers := http.Header{cartSessionHeader: []string{guestSessionToken}}
			actual := StartServer(t, e).DoJSON(http.MethodGet, meCartPath, nil, headers)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			AssertJSONResponseType[gen.CartResponse](t, actual)
			assert.Nil(t, captured.UserID)
			require.NotNil(t, captured.SessionToken)
			assert.Equal(t, guestSessionToken, *captured.SessionToken)
		})

		t.Run("主体が無くても空のカートを200で返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_cartuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetCart(gomock.Any(), cartuc.Subject{}).
				Return(cartuc.CartView{Items: []cartuc.CartItemView{}}, nil)

			carts.BindHandler(e, tf, uc)

			actual := StartServer(t, e).DoJSON(http.MethodGet, meCartPath, nil, nil)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			// 明細が空でも items が null ではなく [] で返ることを AssertJSONResponseType が検証する。
			AssertJSONResponseType[gen.CartResponse](t, actual)
		})

		t.Run("実ミドルウェア経由でも資格情報が無ければ匿名として200を返す", func(t *testing.T) {
			t.Parallel()

			// 任意認証の宣言（BearerAuth と空要件の OR）が、資格情報を持たない呼び出し元を
			// 実際に通すことを確かめる。無効な資格情報を拒否する側は異常系で固定している。
			e := echo.New()
			UseAppErrorHandler(t, e)
			useOpenAPIValidation(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_cartuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetCart(gomock.Any(), cartuc.Subject{}).
				Return(cartuc.CartView{Items: []cartuc.CartItemView{}}, nil)

			carts.BindHandler(e, tf, uc)

			actual := StartServer(t, e).DoJSON(http.MethodGet, meCartPath, nil, nil)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			AssertJSONResponseType[gen.CartResponse](t, actual)
		})

		t.Run("再評価の結果が立った明細を含めて200を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_cartuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetCart(gomock.Any(), gomock.Any()).Return(cartuc.CartView{
				Items: []cartuc.CartItemView{{
					ProductID: uuid.UUID{},
					Quantity:  1,
					Issues:    []cartuc.ItemIssue{cartuc.ItemIssueOutOfStock},
				}},
			}, nil)

			carts.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_ct_issue"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, meCartPath, nil, headers)
			// 買えない明細があってもエラーにしない。4xx にすると買える明細を見せられなくなる。
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			AssertJSONResponseType[gen.CartResponse](t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ErrInternalで500を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_cartuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetCart(gomock.Any(), gomock.Any()).Return(cartuc.CartView{}, apperror.ErrInternal)

			carts.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_ct_err"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, meCartPath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})

		t.Run("無効な資格情報は匿名として通さず401を返す", func(t *testing.T) {
			t.Parallel()

			// 任意認証の operation は資格情報が無ければ匿名で通すが、提示された資格情報の検証に
			// 失敗した場合は拒否する。空の security 要件が必ず満たされるため、この拒否は
			// バリデーションの結果ではなく failClosed が担う（ADR-0019）。
			e := echo.New()
			UseAppErrorHandler(t, e)
			useOpenAPIValidationWithAuthFailure(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_cartuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetCart(gomock.Any(), gomock.Any()).Times(0)

			carts.BindHandler(e, tf, uc)

			headers := http.Header{"Authorization": []string{"Bearer invalid-token"}}
			actual := StartServer(t, e).DoJSON(http.MethodGet, meCartPath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusUnauthorized)
		})

		t.Run("形式が不正なX-Cart-Sessionは400で弾かれUsecaseは呼ばれない", func(t *testing.T) {
			t.Parallel()

			cases := map[string]string{
				"42文字（短い）":    "abcdefghijklmnopqrstuvwxyz0123456789-_ABCD",
				"44文字（長い）":    "abcdefghijklmnopqrstuvwxyz0123456789-_ABCDEF",
				"URL-safeでない": "abcdefghijklmnopqrstuvwxyz0123456789-_ABCD+",
			}
			for name, token := range cases {
				t.Run(name, func(t *testing.T) {
					t.Parallel()

					e := echo.New()
					UseAppErrorHandler(t, e)
					useOpenAPIValidation(t, e)
					ctrl := gomock.NewController(t)
					tf := observability.NewNoopTracerFactory(t)

					uc := mock_cartuc.NewMockUsecase(ctrl)
					uc.EXPECT().GetCart(gomock.Any(), gomock.Any()).Times(0)

					carts.BindHandler(e, tf, uc)

					headers := http.Header{cartSessionHeader: []string{token}}
					actual := StartServer(t, e).DoJSON(http.MethodGet, meCartPath, nil, headers)
					AssertErrorResponse(t, actual, http.StatusBadRequest)
				})
			}
		})

		t.Run("POSTは405を返しUsecaseは呼ばれない", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_cartuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetCart(gomock.Any(), gomock.Any()).Times(0)

			carts.BindHandler(e, tf, uc)

			actual := StartServer(t, e).DoJSON(http.MethodPost, meCartPath, nil, nil)
			AssertErrorResponse(t, actual, http.StatusMethodNotAllowed)
		})
	})
}
