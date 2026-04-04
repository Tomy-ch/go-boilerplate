package integration

import (
	"net/http"
	"testing"

	"go-boilerplate/internal/controller/handler/health/gen"
	"go-boilerplate/internal/controller/handler/healthz"
	"go-boilerplate/internal/observability"

	"github.com/labstack/echo/v4"
)

func TestHealthz_Integration(t *testing.T) {
	t.Parallel()

	t.Run("GET /healthzのエンドポイントが正常に動作することを確認する", func(t *testing.T) {
		e := echo.New()
		tf := observability.NewNoopTracerFactory(t)

		healthz.BindHandler(e, tf)

		actual := StartServer(t, e).DoJSON(http.MethodGet, "/healthz", nil, nil)
		AssertJSONResponse(t, gen.HealthResponse{}, actual)
	})
}
