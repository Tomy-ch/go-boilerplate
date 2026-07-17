package healthz

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/controller/handler/healthz/gen"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	"go-boilerplate/internal/observability"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const targetPath = "/healthz"

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

func Test_server_GetHealthz(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ステータスokを返す", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			s := &server{tracer: observability.NewMockControllerLayerTracer(t)}
			expectedResponse := gen.HealthResponse{Status: "ok"}

			resp, err := s.GetHealthz(ctx, gen.GetHealthzRequestObject{})
			require.NoError(t, err)

			actual, ok := resp.(gen.GetHealthz200JSONResponse)
			require.True(t, ok)

			assert.Equal(t, expectedResponse, gen.HealthResponse(actual))
		})
	})
}

func TestGetHealthz_OverHTTP(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)
	BindHandler(e, tf)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, targetPath, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	testassert.AssertJSONEqual(t, http.StatusOK, gen.HealthResponse{Status: "ok"}, rec)
}
