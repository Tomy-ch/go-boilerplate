package server

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestModule(t *testing.T) {
	t.Parallel()

	t.Run("Module が nil でないこと", func(t *testing.T) {
		t.Parallel()
		opt := Module()
		require.NotNil(t, opt)
	})

	t.Run("ServerConfig を供給すると NewAppServer が実行され *echo.Echo が構築されること", func(t *testing.T) {
		t.Parallel()

		var got *echo.Echo
		app := fx.New(
			fx.NopLogger,
			Module(),
			fx.Supply(config.NewServerConfig(config.MockConfigForTest(t))),
			fx.Populate(&got),
		)
		require.NoError(t, app.Err())
		require.NoError(t, app.Start(context.Background()))
		t.Cleanup(func() { _ = app.Stop(context.Background()) })
		assert.NotNil(t, got)
	})
}

func TestHookModule(t *testing.T) {
	t.Parallel()

	t.Run("HookModule が nil でないこと", func(t *testing.T) {
		t.Parallel()
		opt := HookModule()
		require.NotNil(t, opt)
	})

	t.Run("HookModule を fx.New に渡せること", func(t *testing.T) {
		t.Parallel()
		app := fx.New(HookModule())
		require.NotNil(t, app)
	})
}

func TestMiddlewareModule(t *testing.T) {
	t.Parallel()

	t.Run("MiddlewareModule が nil でないこと", func(t *testing.T) {
		t.Parallel()
		opt := MiddlewareModule()
		require.NotNil(t, opt)
	})

	t.Run("MiddlewareModule を fx.New に渡せること", func(t *testing.T) {
		t.Parallel()
		app := fx.New(MiddlewareModule())
		require.NotNil(t, app)
	})
}
