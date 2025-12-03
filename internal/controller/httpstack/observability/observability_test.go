package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/observability"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	mw := Middleware(appCfg)
	require.NotNil(t, mw)
}

func TestMiddleware_Integration(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)

	e := echo.New()
	e.Use(Middleware(appCfg))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.NotNil(t, rec)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTracerProvider(t *testing.T) {
	t.Parallel()

	var tpProvided any
	app := fx.New(
		fx.Invoke(func(lc fx.Lifecycle) {
			tpProvided = observability.TracerProvider(lc)
		}),
	)

	ctx := context.Background()
	require.NoError(t, app.Start(ctx))
	require.NotNil(t, tpProvided)
	require.NoError(t, app.Stop(ctx))
}
