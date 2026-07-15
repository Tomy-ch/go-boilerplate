package module

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di/lifecycle"
	mock_lifecycle "go-boilerplate/internal/di/lifecycle/mock"
	"go-boilerplate/internal/infrastructure/httpclient"
	"go-boilerplate/internal/logging"
	mock_logging "go-boilerplate/internal/logging/mock"
	infrasystem "go-boilerplate/internal/system"
)

func Test_httpClientModule_ProvidesClient(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("実モジュール配線(clock + httpclient)から Client が構築される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockReg := mock_lifecycle.NewMockRegistrar(ctrl)
			mockLog := mock_logging.NewMockLogger(ctrl)
			mockLF := mock_logging.NewMockLogFieldBuilder(ctrl)
			mockReg.EXPECT().RegisterStop(gomock.Any()).AnyTimes()

			var client httpclient.Client

			// 本番と同じ clockModule()/httpClientModule() を通し、配線そのものを検証する。
			app := fx.New(
				ObservabilityModule(),
				clockModule(),
				httpClientModule(),
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
				fx.Provide(infrasystem.NewBuildInfo),
				fx.Populate(&client),
				fx.NopLogger,
			)

			require.NoError(t, app.Start(context.Background()))
			t.Cleanup(func() { require.NoError(t, app.Stop(context.Background())) })
			assert.NotNil(t, client)
		})
	})
}
