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
	"go-boilerplate/internal/infrastructure/httpclient"
	"go-boilerplate/internal/infrastructure/system"
	"go-boilerplate/internal/logging"
	mock_logging "go-boilerplate/internal/logging/mock"
	infrasystem "go-boilerplate/internal/system"
)

func TestHTTPClientModule_ProvidesClient(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("fx アプリで Client が提供される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockReg := mock_lifecycle.NewMockRegistrar(ctrl)
			mockLog := mock_logging.NewMockLogger(ctrl)
			mockLF := mock_logging.NewMockLogFieldBuilder(ctrl)
			mockReg.EXPECT().RegisterStop(gomock.Any()).AnyTimes()

			var client httpclient.Client

			app := fx.New(
				ObservabilityModule(),
				fx.Provide(func() testing.TB { return t }),
				fx.Provide(func() lifecycle.Registrar { return mockReg }),
				fx.Provide(func() logging.Logger { return mockLog }),
				fx.Provide(func() logging.LogFieldBuilder { return mockLF }),
				fx.Provide(func() *config.ApplicationConfig {
					return config.NewApplicationConfig(config.MockConfigForTest(t))
				}),
				fx.Provide(infrasystem.NewBuildInfo),
				fx.Provide(
					system.NewSleeper,
					httpclient.NewDefaultRegistry,
					httpclient.New,
				),
				fx.Populate(&client),
				fx.NopLogger,
			)

			require.NoError(t, app.Start(context.Background()))
			require.NotNil(t, client)
			require.NoError(t, app.Stop(context.Background()))
		})
	})
}
