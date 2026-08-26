package integration

import (
	"net/http"
	"testing"

	"go-boilerplate/internal/controller/handler/health"
	"go-boilerplate/internal/controller/handler/health/gen"
	"go-boilerplate/internal/observability"

	"github.com/labstack/echo/v5"
)

func TestHealth_Integration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GET /healthがHealthResponseを返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			tf := observability.NewNoopTracerFactory(t)

			health.BindHandler(e, tf)

			actual := StartServer(t, e).DoJSON(http.MethodGet, "/health", nil, nil)
			AssertJSONResponseType[gen.HealthResponse](t, actual)
		})
	})
}
