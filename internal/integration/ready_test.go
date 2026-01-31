package integration

import (
	"net/http"
	"testing"

	"boilerplate-go/internal/controller/handler/ready"
	"boilerplate-go/internal/observability"
	"boilerplate-go/internal/usecase/healthcheck"
	mock_healthcheck "boilerplate-go/internal/usecase/healthcheck/mock"

	"github.com/labstack/echo/v4"
	"go.uber.org/mock/gomock"
)

func TestReady_Integration(t *testing.T) {
	t.Parallel()

	t.Run("GET /readyのエンドポイントが正常に動作することを確認する", func(t *testing.T) {
		e := echo.New()
		ctrl := gomock.NewController(t)
		tf := observability.NewNoopTracerFactory(t)

		mockApp := mock_healthcheck.NewMockUsecase(ctrl)
		mockApp.EXPECT().CheckHealth(gomock.Any()).Return(healthcheck.DTO{}, nil)

		ready.BindHandler(e, tf, mockApp)
		actual := StartServer(t, e).DoJSON(http.MethodGet, "/ready", nil, nil)
		AssertJSONResponse(t, healthcheck.DTO{}, actual)
	})
}
