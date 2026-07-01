package integration

import (
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	exchangeratehandler "go-boilerplate/internal/controller/handler/v1/exchangerate"
	"go-boilerplate/internal/controller/handler/v1/exchangerate/gen"
	"go-boilerplate/internal/observability"
	mock_exchangerate "go-boilerplate/internal/usecase/exchangerate/mock"

	"github.com/labstack/echo/v4"
	"go.uber.org/mock/gomock"
)

func TestV1ExchangeRatesIntegration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/exchange-rates が ExchangeRateResponse を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_exchangerate.NewMockUsecase(ctrl)
			mockUC.EXPECT().
				Convert(gomock.Any(), "USD", "JPY", 100.0).
				Return(15050.0, nil)

			exchangeratehandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(
				http.MethodGet, "/v1/exchange-rates?base=USD&quote=JPY&amount=100", nil, nil,
			)
			AssertJSONResponseType[gen.ExchangeRateResponse](t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /v1/exchange-rates が ErrUnavailable で 503 を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			// gateway が外部為替サービス不通で ErrUnavailable を返す実挙動を模擬する。
			mockUC := mock_exchangerate.NewMockUsecase(ctrl)
			mockUC.EXPECT().
				Convert(gomock.Any(), "USD", "JPY", 100.0).
				Return(0.0, apperror.ErrUnavailable)

			exchangeratehandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(
				http.MethodGet, "/v1/exchange-rates?base=USD&quote=JPY&amount=100", nil, nil,
			)
			AssertErrorResponse(t, actual, http.StatusServiceUnavailable)
		})
	})
}
