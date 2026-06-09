package health

import (
	"context"
	"net/http"
	"testing"

	"go-boilerplate/internal/controller/handler/health/gen"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	"go-boilerplate/internal/observability"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const targetPath = "/health"

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)
	BindHandler(e, tf)

	expectedMethods := []string{http.MethodGet}
	testassert.AssertEchoRouterPath(
		t, targetPath, e.Routes(),
	)
	testassert.AssertEchoRouterMethods(
		t, expectedMethods, e.Routes(),
	)
}

func TestGetHealth(t *testing.T) {
	t.Parallel()

	t.Run("正常系_ステータスokを返す", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		s := &server{
			tracer: observability.NewMockControllerLayerTracer(t),
		}

		expectedResponse := gen.HealthResponse{Status: "ok"}

		resp, err := s.GetHealth(ctx, gen.GetHealthRequestObject{})
		require.NoError(t, err)

		actual, ok := resp.(gen.GetHealth200JSONResponse)
		require.True(t, ok)

		assert.Equal(t, expectedResponse, gen.HealthResponse(actual))
	})
}
