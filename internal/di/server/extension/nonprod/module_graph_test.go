package nonprod

import (
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/di/server/extension/testkit"

	"go.uber.org/fx"
)

func TestDebugModeModule_ProvidesServeConfig(t *testing.T) {
	t.Parallel()
	testkit.RequireProvidesOne[extension.SrvCfg](t, "server.configurators",
		DebugModeModule(),
		fx.Supply(config.NewApplicationConfig(config.MockConfigForTest(t))),
	)
}
