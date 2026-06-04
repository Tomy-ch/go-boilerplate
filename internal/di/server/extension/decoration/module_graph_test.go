package decoration

import (
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/di/server/extension/testkit"

	"go.uber.org/fx"
)

func TestBannerModule_ProvidesServeConfig(t *testing.T) {
	t.Parallel()
	testkit.RequireProvidesOne[extension.SrvCfg](t, "server.configurators",
		BannerModule(),
		fx.Supply(config.NewApplicationConfig(config.MockConfigForTest(t))),
	)
}

func TestDefaultPortModule_ProvidesServeConfig(t *testing.T) {
	t.Parallel()
	testkit.RequireProvidesOne[extension.SrvCfg](t, "server.configurators",
		DefaultPortModule(),
		fx.Supply(config.NewApplicationConfig(config.MockConfigForTest(t))),
	)
}
