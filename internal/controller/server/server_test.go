package server

import (
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/middleware/binder"
	"boilerplate-go/internal/controller/middleware/ipextractor"
	"boilerplate-go/internal/controller/middleware/validator"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	defaultEcho := echo.New()
	httpErrorHandler := defaultEcho.HTTPErrorHandler
	validator := validator.NewValidator()
	binder := binder.NewBinder()
	ipextractor := ipextractor.NewIPExtractor(cfg)

	e := New(cfg, validator, binder, ipextractor, httpErrorHandler)

	require.NotNil(t, e)
	require.Equal(t, validator, e.Validator)
	require.Equal(t, binder, e.Binder)
	require.NotNil(t, e.IPExtractor)
	require.NotNil(t, e.HTTPErrorHandler)
}

func TestSetPrimitiveEchoSettings(t *testing.T) {
	t.Parallel()
	t.Run("本番環境モード", func(t *testing.T) {
		t.Parallel()

		e := echo.New()

		cfg := config.MockConfigForTest(t)
		cfg.SetServerAppMode(t, config.ProductionMode)

		setPrimitiveEchoSettings(e, &cfg)

		require.False(t, e.Debug)
		require.True(t, e.HideBanner)
		require.True(t, e.HidePort)
	})

	t.Run("開発環境モード", func(t *testing.T) {
		t.Parallel()

		e := echo.New()

		cfg := config.MockConfigForTest(t)
		cfg.SetServerAppMode(t, config.DevelopmentMode)

		setPrimitiveEchoSettings(e, &cfg)

		require.True(t, e.Debug)
		require.False(t, e.HideBanner)
		require.False(t, e.HidePort)
	})
}

func TestSetCustomEchoBindings(t *testing.T) {
	t.Parallel()
	e := echo.New()
	httpErrorHandler := echo.New().HTTPErrorHandler
	validator := validator.NewValidator()
	binder := binder.NewBinder()
	ipextractor := ipextractor.NewIPExtractor(&config.Config{})

	setCustomEchoBindings(e, validator, binder, ipextractor, httpErrorHandler)

	require.Equal(t, validator, e.Validator)
	require.Equal(t, binder, e.Binder)
	require.NotNil(t, e.IPExtractor)
	require.NotNil(t, e.HTTPErrorHandler)
}
