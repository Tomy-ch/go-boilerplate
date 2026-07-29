package module

import (
	"context"
	"testing"

	"github.com/exaring/otelpgx"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di/lifecycle"
	mock_lifecycle "go-boilerplate/internal/di/lifecycle/mock"
	"go-boilerplate/internal/logging"
	mock_logging "go-boilerplate/internal/logging/mock"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/system"
)

func TestObservabilityModule_ProvidesTracerFactory(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("fx アプリで TracerFactory が提供される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockReg := mock_lifecycle.NewMockRegistrar(ctrl)
			mockLog := mock_logging.NewMockLogger(ctrl)
			mockLF := mock_logging.NewMockLogFieldBuilder(ctrl)

			// ProviderShutdowner の Shutdown が単一の Stop フックとして登録される。
			mockReg.EXPECT().RegisterStop(gomock.Any()).Times(1)

			// buildinfo.Register はデフォルトレジストリへ登録するため、テスト間の汚染を避ける。
			origReg := prometheus.DefaultRegisterer
			prometheus.DefaultRegisterer = prometheus.NewRegistry()
			t.Cleanup(func() { prometheus.DefaultRegisterer = origReg })

			var tf observability.TracerFactory

			app := fx.New(
				ObservabilityModule(),
				fx.Provide(func() testing.TB { return t }),
				fx.Provide(func() lifecycle.Registrar { return mockReg }),
				fx.Provide(func() logging.Logger { return mockLog }),
				fx.Provide(func() logging.LogFieldBuilder { return mockLF }),
				fx.Provide(func() *config.ApplicationConfig {
					return config.NewApplicationConfig(config.MockConfigForTest(t))
				}),
				fx.Provide(func() *config.ObservabilityConfig {
					oc := config.NewObservabilityConfig(config.MockConfigForTest(t))
					oc.SetObservabilityTracesExporter(t, "")
					oc.SetObservabilityMetricsExporter(t, "")
					oc.SetObservabilityLogsExporter(t, "")
					return oc
				}),
				fx.Provide(system.NewBuildInfo),
				fx.Populate(&tf),
				fx.NopLogger,
			)

			require.NoError(t, app.Start(context.Background()))
			t.Cleanup(func() { require.NoError(t, app.Stop(context.Background())) })
			assert.NotNil(t, tf)
		})
	})
}

func TestObservabilityModule(t *testing.T) {
	t.Parallel()

	// ObservabilityModule は config / logging / BuildInfo を外から受け取るため、
	// 依存だけを供給した最小構成に対象モジュールを重ねて検証する。
	obsDeps := func(t *testing.T) []fx.Option {
		t.Helper()

		ctrl := gomock.NewController(t)
		mockReg := mock_lifecycle.NewMockRegistrar(ctrl)
		mockLog := mock_logging.NewMockLogger(ctrl)
		mockLF := mock_logging.NewMockLogFieldBuilder(ctrl)

		return []fx.Option{
			fx.Provide(func() testing.TB { return t }),
			fx.Provide(func() lifecycle.Registrar { return mockReg }),
			fx.Provide(func() logging.Logger { return mockLog }),
			fx.Provide(func() logging.LogFieldBuilder { return mockLF }),
			fx.Provide(func() *config.ApplicationConfig {
				return config.NewApplicationConfig(config.MockConfigForTest(t))
			}),
			fx.Provide(func() *config.ObservabilityConfig {
				return config.NewObservabilityConfig(config.MockConfigForTest(t))
			}),
			fx.Provide(system.NewBuildInfo),
			fx.NopLogger,
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("logging が要求する LogCore と TraceExtractor の seam を提供する", func(t *testing.T) {
			t.Parallel()

			var (
				core    logging.LogCore
				extract logging.TraceExtractor
			)

			opts := append(obsDeps(t), ObservabilityModule(), fx.Populate(&core, &extract))
			require.NoError(t, fx.ValidateApp(opts...))
		})

		t.Run("DB / HTTP client 計装が要求する pgx tracer と transport を提供する", func(t *testing.T) {
			t.Parallel()

			var (
				pgxTracer *otelpgx.Tracer
				transport *observability.HTTPClientTransport
			)

			opts := append(obsDeps(t), ObservabilityModule(), fx.Populate(&pgxTracer, &transport))
			require.NoError(t, fx.ValidateApp(opts...))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未配線では TracerFactory が解決できずグラフ検証に失敗する", func(t *testing.T) {
			t.Parallel()

			var tf observability.TracerFactory

			opts := append(obsDeps(t), fx.Populate(&tf))
			require.Error(t, fx.ValidateApp(opts...))
		})
	})
}
