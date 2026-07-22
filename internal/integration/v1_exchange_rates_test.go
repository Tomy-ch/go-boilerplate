package integration

import (
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	exchangeratehandler "go-boilerplate/internal/controller/handler/v1/exchangerate"
	"go-boilerplate/internal/controller/handler/v1/exchangerate/gen"
	"go-boilerplate/internal/observability"
	exchangerateuc "go-boilerplate/internal/usecase/exchangerate"
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
				Convert(gomock.Any(), exchangerateuc.ConvertInput{Base: "USD", Quote: "JPY", Amount: 100}).
				Return(&exchangerateuc.ConvertResult{Converted: 15050}, nil)

			exchangeratehandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(
				http.MethodGet, "/v1/exchange-rates?base=USD&quote=JPY&amount=100", nil, nil,
			)
			AssertJSONResponseType[gen.ExchangeRateResponse](t, actual)
		})

		t.Run("displayCurrency=JPY 指定でも 200 で ExchangeRateResponse を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockUC := mock_exchangerate.NewMockUsecase(ctrl)
			mockUC.EXPECT().
				Convert(gomock.Any(), gomock.Any()).
				Return(&exchangerateuc.ConvertResult{
					Converted: 15050,
					Reference: &exchangerateuc.ReferenceAmount{
						Currency: "JPY", Amount: 15050, Rate: 150.5, RateDate: "2026-07-21",
					},
				}, nil)

			exchangeratehandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(
				http.MethodGet,
				"/v1/exchange-rates?base=USD&quote=JPY&amount=100&displayCurrency=JPY", nil, nil,
			)
			AssertJSONResponseType[gen.ExchangeRateResponse](t, actual)
		})

		t.Run("degrade: 参考換算のレート取得失敗でも本体は 200 で継続する", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			// 参考換算のレート取得だけが失敗し reference=nil の degrade を模擬する。本体は 503 でなく 200。
			mockUC := mock_exchangerate.NewMockUsecase(ctrl)
			mockUC.EXPECT().
				Convert(gomock.Any(), gomock.Any()).
				Return(&exchangerateuc.ConvertResult{Converted: 15050, Reference: nil}, nil)

			exchangeratehandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(
				http.MethodGet,
				"/v1/exchange-rates?base=USD&quote=JPY&amount=100&displayCurrency=JPY", nil, nil,
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

			// 本体換算のレート取得が外部為替サービス不通で失敗する実挙動を模擬する。
			mockUC := mock_exchangerate.NewMockUsecase(ctrl)
			mockUC.EXPECT().
				Convert(gomock.Any(), gomock.Any()).
				Return(nil, apperror.ErrUnavailable)

			exchangeratehandler.BindHandler(e, tf, mockUC)

			actual := StartServer(t, e).DoJSON(
				http.MethodGet, "/v1/exchange-rates?base=USD&quote=JPY&amount=100", nil, nil,
			)
			AssertErrorResponse(t, actual, http.StatusServiceUnavailable)
		})
	})
}
