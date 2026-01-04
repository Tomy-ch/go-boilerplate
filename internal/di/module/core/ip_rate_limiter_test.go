package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/httpstack/ratelimit"
)

func TestIPRateLimiterModule_ProvidesIPRateLimiter(t *testing.T) {
	t.Run("fx アプリで IPRateLimiter が提供される", func(t *testing.T) {
		var rl ratelimit.IPRateLimiter
		app := fx.New(
			IPRateLimiterModule(),
			fx.Provide(func() testing.TB { return t }),
			fx.Provide(config.MockConfigForTest),
			fx.Provide(config.NewIPRateLimitConfig),
			fx.Populate(&rl),
			fx.NopLogger,
		)

		require.NoError(t, app.Start(context.Background()))
		require.NotNil(t, rl)
		require.NotPanics(t, func() { rl.Cleanup() })
		require.NoError(t, app.Stop(context.Background()))
	})
}
