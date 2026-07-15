package instrumentation

import (
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/di/server/extension/testkit"
	"go-boilerplate/internal/logging"

	"go.uber.org/fx"
)

func TestLoggingModule(t *testing.T) {
	t.Parallel()
	testkit.RequireProvidesOne[extension.UseMiddleware](t, "middlewares.use",
		LoggingModule(),
		fx.Provide(func() logging.Logger { return logging.NewTestLogger(t) }),
		fx.Provide(func() logging.LogFieldBuilder { return logging.NewTestLogFieldBuilder(t) }),
	)
}

func TestObservabilityModule(t *testing.T) {
	t.Parallel()
	cfg := config.MockConfigForTest(t)
	testkit.RequireProvidesOne[extension.UseMiddleware](t, "middlewares.use",
		ObservabilityModule(),
		fx.Supply(config.NewApplicationConfig(cfg)),
		fx.Supply(config.NewObservabilityConfig(cfg)),
	)
}

func TestRequestIDModule(t *testing.T) {
	t.Parallel()
	testkit.RequireProvidesOne[extension.UseMiddleware](t, "middlewares.use", RequestIDModule())
}
