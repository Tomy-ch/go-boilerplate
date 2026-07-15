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

// TestServerShutdownValidationWiring は、config.ValidateServerShutdown が server グラフの起動時に
// fx.Invoke 経由で適用されること（＝結線が機能すること）を検証する。値の妥当性ロジック自体の
// 境界検証は config 側（Test_validateServerShutdown）が担う。
//
//nolint:paralleltest // config.New が env(EnsureRepoRootAndEnv/t.Setenv/t.Chdir)を使うため並列化不可
func TestServerShutdownValidationWiring(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("shutdown>=request の設定では fx.Invoke 経由でも起動できる", func(t *testing.T) {
			cfg := config.MockConfigForTest(t)
			app := fx.New(
				fx.NopLogger,
				fx.Supply(config.NewApplicationConfig(cfg), config.NewServerConfig(cfg)),
				fx.Invoke(config.ValidateServerShutdown),
			)
			require.NoError(t, app.Start(context.Background()))
			require.NoError(t, app.Stop(context.Background()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("shutdown<request の設定では fx.Invoke が起動を失敗させる", func(t *testing.T) {
			config.EnsureRepoRootAndEnv(t, config.TestingEnvValue)
			t.Setenv("SERVER_REQUEST_TIMEOUT", "60s")
			t.Setenv("APP_SHUTDOWN_TIMEOUT", "30s")

			cfg, err := config.New()
			require.NoError(t, err) // New() は交差検証しないため、不正設定でも構築は成功する

			app := fx.New(
				fx.NopLogger,
				fx.Supply(config.NewApplicationConfig(cfg), config.NewServerConfig(cfg)),
				fx.Invoke(config.ValidateServerShutdown),
			)
			require.Error(t, app.Start(context.Background()))
		})
	})
}
