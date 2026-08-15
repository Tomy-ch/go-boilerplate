package integration

import (
	"context"
	"net/http"
	"testing"

	mergehandler "go-boilerplate/internal/controller/handler/v1/carts/merge"
	"go-boilerplate/internal/controller/handler/v1/carts/merge/gen"
	"go-boilerplate/internal/observability"
	cartuc "go-boilerplate/internal/usecase/cart"
	mock_cartuc "go-boilerplate/internal/usecase/cart/mock"
	"go-boilerplate/internal/usecase/idempotency"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// meCartMergePath は、引き継ぎのエンドポイントのパス。
const meCartMergePath = "/v1/carts/me/merge"

// echoWithMergeHandler は、モックしたユースケースを結線した Echo を返します。
func echoWithMergeHandler(t *testing.T, expect func(*mock_cartuc.MockUsecase)) *echo.Echo {
	t.Helper()

	e := echo.New()
	uc := mock_cartuc.NewMockUsecase(gomock.NewController(t))
	expect(uc)
	mergehandler.BindHandler(e, observability.NewNoopTracerFactory(t), uc, idempotency.Deps{})
	return e
}

func TestV1CartsMeMerge_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証済みで引き継ぐと失われた分がCartMergeResponseで返る", func(t *testing.T) {
			t.Parallel()

			clamped := uuidtestkit.NewTestFromSalt(t, "int_mg_clamped")
			e := echoWithMergeHandler(t, func(uc *mock_cartuc.MockUsecase) {
				uc.EXPECT().MergeOnLogin(gomock.Any(), gomock.Any()).
					Return(cartuc.MergeCartResult{Clamped: []uuid.UUID{clamped}}, nil)
			})

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_mg_user"))
			headers.Set(cartSessionHeader, guestSessionToken)
			actual := StartServer(t, e).DoJSON(http.MethodPost, meCartMergePath, nil, headers)

			assert.Equal(t, http.StatusOK, actual.StatusCode)
			AssertJSONResponseType[gen.CartMergeResponse](t, actual)
		})

		t.Run("引き継ぎ元のトークンがユースケースへ渡る", func(t *testing.T) {
			t.Parallel()

			var captured cartuc.MergeOnLoginParams
			userID := uuidtestkit.NewTestFromSalt(t, "int_mg_owner")
			e := echoWithMergeHandler(t, func(uc *mock_cartuc.MockUsecase) {
				uc.EXPECT().MergeOnLogin(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, params cartuc.MergeOnLoginParams) (cartuc.MergeCartResult, error) {
						captured = params
						return cartuc.MergeCartResult{}, nil
					},
				)
			})

			headers := MakeAvailableUserID(t, e, userID)
			headers.Set(cartSessionHeader, guestSessionToken)
			actual := StartServer(t, e).DoJSON(http.MethodPost, meCartMergePath, nil, headers)

			assert.Equal(t, http.StatusOK, actual.StatusCode)
			assert.Equal(t, userID, captured.UserID)
			assert.Equal(t, guestSessionToken, captured.SessionToken)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未認証は401を返しUsecaseは呼ばれない", func(t *testing.T) {
			t.Parallel()

			// この operation だけ認証必須で、任意認証の 4 本とは security の宣言が異なる。
			// 認証失敗を強制するヘルパーは使わない。それは spec の security 検査を先に踏ませるため、
			// 認証必須の op では 403 になり、本番の経路（ハンドラが主体の不在を検出して 401）と違う。
			e := echoWithMergeHandler(t, func(uc *mock_cartuc.MockUsecase) {
				uc.EXPECT().MergeOnLogin(gomock.Any(), gomock.Any()).Times(0)
			})
			UseAppErrorHandler(t, e)

			headers := http.Header{cartSessionHeader: []string{guestSessionToken}}
			actual := StartServer(t, e).DoJSON(http.MethodPost, meCartMergePath, nil, headers)

			AssertErrorResponse(t, actual, http.StatusUnauthorized)
		})

		t.Run("X-Cart-Sessionが無ければ400で弾かれUsecaseは呼ばれない", func(t *testing.T) {
			t.Parallel()

			// 引き継ぎ元を指定しない引き継ぎは要求として成り立たないため、ヘッダは必須。
			e := echoWithMergeHandler(t, func(uc *mock_cartuc.MockUsecase) {
				uc.EXPECT().MergeOnLogin(gomock.Any(), gomock.Any()).Times(0)
			})
			UseAppErrorHandler(t, e)
			useOpenAPIValidation(t, e)

			actual := StartServer(t, e).DoJSON(http.MethodPost, meCartMergePath, nil, nil)

			AssertErrorResponse(t, actual, http.StatusBadRequest)
		})

		t.Run("ユースケースのエラーは応答へ写る", func(t *testing.T) {
			t.Parallel()

			e := echoWithMergeHandler(t, func(uc *mock_cartuc.MockUsecase) {
				uc.EXPECT().MergeOnLogin(gomock.Any(), gomock.Any()).
					Return(cartuc.MergeCartResult{}, cartuc.ErrUnavailableProduct)
			})
			UseAppErrorHandler(t, e)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_mg_err"))
			headers.Set(cartSessionHeader, guestSessionToken)
			actual := StartServer(t, e).DoJSON(http.MethodPost, meCartMergePath, nil, headers)

			AssertErrorResponse(t, actual, http.StatusUnprocessableEntity)
		})

		t.Run("GETは405を返しUsecaseは呼ばれない", func(t *testing.T) {
			t.Parallel()

			e := echoWithMergeHandler(t, func(uc *mock_cartuc.MockUsecase) {
				uc.EXPECT().MergeOnLogin(gomock.Any(), gomock.Any()).Times(0)
			})
			UseAppErrorHandler(t, e)

			actual := StartServer(t, e).DoJSON(http.MethodGet, meCartMergePath, nil, nil)

			AssertErrorResponse(t, actual, http.StatusMethodNotAllowed)
		})
	})
}
