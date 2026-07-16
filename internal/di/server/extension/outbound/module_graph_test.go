package outbound

import (
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/oapi/validator"
	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/di/server/extension/testkit"
	"go-boilerplate/internal/logging"

	"go.uber.org/fx"
)

func TestErrorHandlerModule(t *testing.T) {
	t.Parallel()
	spec, err := validator.GetValidator()
	if err != nil {
		t.Fatalf("failed to load OpenAPI spec: %v", err)
	}
	testkit.RequireProvidesOne[extension.SrvCfg](t, "server.configurators",
		ErrorHandlerModule(),
		fx.Provide(func() logging.Logger { return logging.NewTestLogger(t) }),
		fx.Provide(func() logging.LogFieldBuilder { return logging.NewTestLogFieldBuilder(t) }),
		fx.Supply(config.NewObservabilityConfig(config.MockConfigForTest(t))),
		fx.Supply(spec),
	)
}

func TestForceJSONModule(t *testing.T) {
	t.Parallel()
	testkit.RequireProvidesOne[extension.UseMiddleware](t, "middlewares.use", ForceJSONModule())
}

func TestRecoveryModule(t *testing.T) {
	t.Parallel()
	testkit.RequireProvidesOne[extension.UseMiddleware](t, "middlewares.use",
		RecoveryModule(),
		fx.Provide(func() logging.Logger { return logging.NewTestLogger(t) }),
		fx.Provide(func() logging.LogFieldBuilder { return logging.NewTestLogFieldBuilder(t) }),
		fx.Supply(config.NewApplicationConfig(config.MockConfigForTest(t))),
	)
}
