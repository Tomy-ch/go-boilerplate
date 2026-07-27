package integration

import (
	"net/http"
	"testing"

	"go-boilerplate/internal/controller/handler/ready"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/healthcheck"
	mock_healthcheck "go-boilerplate/internal/usecase/healthcheck/mock"

	"github.com/labstack/echo/v5"
	"go.uber.org/mock/gomock"
)

func TestReady_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /readyがUsecaseのDTOを返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			mockApp := mock_healthcheck.NewMockUsecase(ctrl)
			mockApp.EXPECT().CheckHealth(gomock.Any()).Return(&healthcheck.DTO{}, nil)

			ready.BindHandler(e, tf, mockApp)
			actual := StartServer(t, e).DoJSON(http.MethodGet, "/ready", nil, nil)
			AssertJSONResponseType[healthcheck.DTO](t, actual)
		})
	})
}
