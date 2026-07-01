package exchangerate

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	"go-boilerplate/internal/controller/handler/v1/exchangerate/gen"
	"go-boilerplate/internal/observability"
	mock_exchangerate "go-boilerplate/internal/usecase/exchangerate/mock"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/v1/exchange-rates"

func newServer(t *testing.T) (*server, *mock_exchangerate.MockUsecase) {
	t.Helper()
	mockUC := mock_exchangerate.NewMockUsecase(gomock.NewController(t))
	return &server{tracer: observability.NewMockControllerLayerTracer(t), uc: mockUC}, mockUC
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)
	mockUC := mock_exchangerate.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, tf, mockUC)

	testassert.AssertEchoRouterPath(t, targetPath, e.Routes())
	testassert.AssertEchoRouterMethods(t, []string{http.MethodGet}, e.Routes())
}

func Test_server_GetExchangeRates(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("usecaseの換算結果をレスポンスへ詰め替える", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().Convert(gomock.Any(), "USD", "JPY", 100.0).Return(15050.0, nil)

			resp, err := s.GetExchangeRates(context.Background(), gen.GetExchangeRatesRequestObject{
				Params: gen.GetExchangeRatesParams{Base: "USD", Quote: "JPY", Amount: 100},
			})

			require.NoError(t, err)

			actual, ok := resp.(gen.GetExchangeRates200JSONResponse)
			require.True(t, ok)

			assert.Equal(t, gen.GetExchangeRates200JSONResponse{
				Base:      "USD",
				Quote:     "JPY",
				Amount:    100,
				Converted: 15050,
			}, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("usecaseのエラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().
				Convert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(0.0, apperror.ErrUnavailable)

			_, err := s.GetExchangeRates(context.Background(), gen.GetExchangeRatesRequestObject{
				Params: gen.GetExchangeRatesParams{Base: "USD", Quote: "JPY", Amount: 100},
			})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}
