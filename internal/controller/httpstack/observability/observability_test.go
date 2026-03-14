package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"boilerplate-go/internal/config"
	mock_lifecycle "boilerplate-go/internal/di/lifecycle/mock"
	"boilerplate-go/internal/observability"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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

	ctx := context.Background()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.NotNil(t, rec)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTracerProvider(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	mockReg := mock_lifecycle.NewMockRegistrar(ctrl)
	var shutdownFunc func(context.Context) error
	dummy := func(context.Context) error { return nil }
	mockReg.EXPECT().RegisterStop(gomock.AssignableToTypeOf(dummy)).Do(func(args ...any) {
		shutdownFunc = args[0].(func(context.Context) error)
	}).Times(1)

	tp := observability.TracerProvider(mockReg)
	require.NotNil(t, tp)
	require.NotNil(t, shutdownFunc)

	require.NoError(t, shutdownFunc(context.Background()))
}
