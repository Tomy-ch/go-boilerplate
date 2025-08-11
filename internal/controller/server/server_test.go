package server

import (
	"os"
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/testutil"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.RunWithTestSetup(m))
}

func TestNew(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	httpErrorHandler := echo.New().HTTPErrorHandler
	validator := NewValidator()
	binder := NewBinder()
	ipextractor := NewIPExtractor(cfg)

	e := New(cfg, validator, binder, ipextractor, httpErrorHandler)

	require.NotNil(t, e)
	require.Equal(t, validator, e.Validator)
	require.Equal(t, binder, e.Binder)
	require.NotNil(t, e.IPExtractor)
	require.NotNil(t, e.HTTPErrorHandler)
}

func TestSetPrimitiveEchoSettings(t *testing.T) {
	// 環境変数を書き換えるため、並列実行は避ける
	t.Run("本番環境モード", func(t *testing.T) {
		e := echo.New()

		err := config.Load()
		require.NoError(t, err)
		t.Setenv("APP_MODE", config.ProductionMode)
		cfg, err := config.New()
		require.NoError(t, err)

		setPrimitiveEchoSettings(e, cfg)

		require.False(t, e.Debug)
		require.True(t, e.HideBanner)
		require.True(t, e.HidePort)
	})

	t.Run("開発環境モード", func(t *testing.T) {
		e := echo.New()

		err := config.Load()
		require.NoError(t, err)
		t.Setenv("APP_MODE", config.DevelopmentMode)
		cfg, err := config.New()
		require.NoError(t, err)

		setPrimitiveEchoSettings(e, cfg)

		require.True(t, e.Debug)
		require.False(t, e.HideBanner)
		require.False(t, e.HidePort)
	})
}

func TestSetCustomEchoBindings(t *testing.T) {
	t.Parallel()
	e := echo.New()
	httpErrorHandler := echo.New().HTTPErrorHandler
	validator := NewValidator()
	binder := NewBinder()
	ipextractor := NewIPExtractor(&config.Config{})

	setCustomEchoBindings(e, validator, binder, ipextractor, httpErrorHandler)

	require.Equal(t, validator, e.Validator)
	require.Equal(t, binder, e.Binder)
	require.NotNil(t, e.IPExtractor)
	require.NotNil(t, e.HTTPErrorHandler)
}
