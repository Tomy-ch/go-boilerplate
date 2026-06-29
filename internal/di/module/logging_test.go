package module

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"
)

func TestLoggingModule_ProvidesLoggerAndFields(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("fx アプリで Logger と LogFieldBuilder が提供される", func(t *testing.T) {
			t.Parallel()

			var lg logging.Logger
			var lf logging.LogFieldBuilder

			app := fx.New(
				LoggingModule(),
				fx.Provide(func() testing.TB { return t }),
				fx.Provide(config.MockConfigForTest),
				fx.Provide(
					config.NewApplicationConfig,
					config.NewObservabilityConfig,
					config.NewOperatingSystemConfig,
				),
				fx.Provide(func() logging.LogCore { return nil }),
				fx.Populate(&lg, &lf),
				fx.NopLogger,
			)

			require.NoError(t, app.Start(context.Background()))
			require.NotNil(t, lg)
			require.NotNil(t, lf)
			// basic smoke: calling logger methods should not panic
			require.NotPanics(t, func() { lg.Info("test") })
			require.NoError(t, app.Stop(context.Background()))
		})
	})
}

func Test_provideLogger(t *testing.T) {
	t.Parallel()

	newAppCfg := func(t *testing.T, mode, level string) *config.ApplicationConfig {
		t.Helper()
		appCfg := config.NewApplicationConfig(&config.Config{})
		appCfg.SetApplicationMode(t, mode)
		appCfg.SetApplicationLogLevel(t, level)
		return appCfg
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("本番モードかつinfoでLoggerを返す", func(t *testing.T) {
			t.Parallel()
			lg, err := provideLogger(newAppCfg(t, config.ProductionMode, "info"), nil)
			require.NoError(t, err)
			require.NotNil(t, lg)
		})

		t.Run("開発モードかつdebugでLoggerを返す", func(t *testing.T) {
			t.Parallel()
			lg, err := provideLogger(newAppCfg(t, config.DevelopmentMode, "debug"), nil)
			require.NoError(t, err)
			require.NotNil(t, lg)
		})

		t.Run("本番モードでもdebug指定でLoggerを返す", func(t *testing.T) {
			t.Parallel()
			lg, err := provideLogger(newAppCfg(t, config.ProductionMode, "debug"), nil)
			require.NoError(t, err)
			require.NotNil(t, lg)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不正なログレベルはエラーを返す", func(t *testing.T) {
			t.Parallel()
			lg, err := provideLogger(newAppCfg(t, config.ProductionMode, "invalid"), nil)
			require.Error(t, err)
			require.Nil(t, lg)
		})

		t.Run("未知のモードはエラーを返す", func(t *testing.T) {
			t.Parallel()
			lg, err := provideLogger(newAppCfg(t, "unknown", "info"), nil)
			require.Error(t, err)
			require.Nil(t, lg)
		})
	})
}
