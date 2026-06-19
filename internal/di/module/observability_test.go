package module

import (
	"context"
	"testing"

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

	t.Run("fx アプリで TracerFactory が提供される", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)

		mockReg := mock_lifecycle.NewMockRegistrar(ctrl)
		mockLog := mock_logging.NewMockLogger(ctrl)
		mockLF := mock_logging.NewMockLogFieldBuilder(ctrl)

		// ProviderShutdowner の Shutdown が単一の Stop フックとして登録される。
		mockReg.EXPECT().RegisterStop(gomock.Any()).Times(1)

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
			fx.Provide(system.NewBuildInfo),
			fx.Populate(&tf),
			fx.NopLogger,
		)

		require.NoError(t, app.Start(context.Background()))
		require.NotNil(t, tf)
		require.NoError(t, app.Stop(context.Background()))
	})
}
