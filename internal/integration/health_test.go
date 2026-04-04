package integration

import (
	"net/http"
	"testing"

	"boilerplate-go/internal/controller/handler/health"
	"boilerplate-go/internal/controller/handler/health/gen"
	"boilerplate-go/internal/observability"

	"github.com/labstack/echo/v4"
)

func TestHealth_Integration(t *testing.T) {
	t.Parallel()

	t.Run("GET /healthのエンドポイントが正常に動作することを確認する", func(t *testing.T) {
		e := echo.New()
		tf := observability.NewNoopTracerFactory(t)

		health.BindHandler(e, tf)

		actual := StartServer(t, e).DoJSON(http.MethodGet, "/health", nil, nil)
		AssertJSONResponse(t, gen.HealthResponse{}, actual)
	})
}
