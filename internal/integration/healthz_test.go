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

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /healthzがHealthResponseを返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			tf := observability.NewNoopTracerFactory(t)

			healthz.BindHandler(e, tf)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/healthz", nil, nil)
			AssertJSONResponseType[gen.HealthResponse](t, actual)
		})
	})
}
