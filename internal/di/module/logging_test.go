package module

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/logging"
)

func TestLoggingModule_ProvidesLoggerAndFields(t *testing.T) {
	t.Run("fx アプリで Logger と LogFieldBuilder が提供される", func(t *testing.T) {
		var lg logging.Logger
		var lf logging.LogFieldBuilder

		app := fx.New(
			LoggingModule(),
			fx.Provide(func() testing.TB { return t }),
			fx.Provide(config.MockConfigForTest),
			fx.Provide(
				config.NewApplicationConfig,
				config.NewObservabilityConfig,
				config.NewOperationSystemConfig,
			),
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
}
