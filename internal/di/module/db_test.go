package module

import (
	"context"
	"testing"

	"github.com/exaring/otelpgx"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/di/server/hook"
	"go-boilerplate/internal/logging"
	mock_logging "go-boilerplate/internal/logging/mock"
	"go-boilerplate/internal/observability"
)

func TestDatabaseModule_Composes(t *testing.T) {
	t.Parallel()

	t.Run("fx アプリに DatabaseModule を追加して起動できる", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)

		mockLogger := mock_logging.NewMockLogger(ctrl)
		mockLF := mock_logging.NewMockLogFieldBuilder(ctrl)

		mockLogger.EXPECT().Named(gomock.Any()).Return(mockLogger).AnyTimes()
		mockLogger.EXPECT().Info("Closing database connection").AnyTimes()

		app := fx.New(
			lifecycle.Module(),
			DatabaseModule(),
			fx.Provide(func() testing.TB { return t }),
			fx.Provide(config.MockConfigForTest),
			fx.Provide(config.NewDatabaseConfig),
			fx.Provide(config.NewObservabilityConfig),
			fx.Provide(config.NewOperatingSystemConfig),
			fx.Provide(config.NewDBConnectionConfig),
			fx.Provide(func() logging.Logger { return mockLogger }),
			fx.Provide(func() logging.LogFieldBuilder { return mockLF }),
			fx.Provide(func() *otelpgx.Tracer { return otelpgx.NewTracer() }),
			fx.Provide(observability.NewNoopTracerFactory),
			fx.NopLogger,
			fx.Invoke(hook.RegisterDBCloseHooks),
		)

		require.NoError(t, app.Start(context.Background()))
		require.NoError(t, app.Stop(context.Background()))
	})
}
