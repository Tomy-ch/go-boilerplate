package module

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
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
				fx.Provide(func() logging.TraceExtractor { return nil }),
				fx.Populate(&lg, &lf),
				fx.NopLogger,
			)

			require.NoError(t, app.Start(context.Background()))
			t.Cleanup(func() { require.NoError(t, app.Stop(context.Background())) })
			assert.NotNil(t, lg)
			assert.NotNil(t, lf)
			// basic smoke: calling logger methods should not panic
			assert.NotPanics(t, func() { lg.Info(context.Background(), "test") })
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
			lg, err := provideLogger(newAppCfg(t, config.ProductionMode, "info"), nil, nil)
			require.NoError(t, err)
			assert.NotNil(t, lg)
		})

		t.Run("開発モードかつdebugでLoggerを返す", func(t *testing.T) {
			t.Parallel()
			lg, err := provideLogger(newAppCfg(t, config.DevelopmentMode, "debug"), nil, nil)
			require.NoError(t, err)
			assert.NotNil(t, lg)
		})

		t.Run("本番モードでもdebug指定でLoggerを返す", func(t *testing.T) {
			t.Parallel()
			lg, err := provideLogger(newAppCfg(t, config.ProductionMode, "debug"), nil, nil)
			require.NoError(t, err)
			assert.NotNil(t, lg)
		})

		t.Run("TraceExtractorが返却Loggerへ注入され出力時に呼ばれる", func(t *testing.T) {
			t.Parallel()

			invoked := false
			extract := logging.TraceExtractor(func(context.Context) (string, string, bool) {
				invoked = true
				return "", "", false
			})

			// 出力レベルを error にし、Debug で probe する。Debug エントリは zap に破棄され
			// stdout を汚さないが、trace 注入（=extract 呼び出し）は引数評価で必ず走るため、
			// extract が返却 Logger へ配線されていることを副作用なく検証できる。
			lg, err := provideLogger(newAppCfg(t, config.ProductionMode, "error"), nil, extract)
			require.NoError(t, err)
			require.NotNil(t, lg)

			lg.Debug(context.Background(), "probe")
			assert.True(t, invoked)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不正なログレベルはエラーを返す", func(t *testing.T) {
			t.Parallel()
			lg, err := provideLogger(newAppCfg(t, config.ProductionMode, "invalid"), nil, nil)
			require.Error(t, err)
			assert.Nil(t, lg)
		})

		t.Run("未知のモードはエラーを返す", func(t *testing.T) {
			t.Parallel()
			lg, err := provideLogger(newAppCfg(t, "unknown", "info"), nil, nil)
			require.Error(t, err)
			assert.Nil(t, lg)
		})
	})
}
