package inbound

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/di/server/extension/testkit"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	echomw "github.com/labstack/echo/v4/middleware"
	"go.uber.org/fx"
)

func TestIPExtractorModule(t *testing.T) {
	t.Parallel()
	cfg := config.MockConfigForTest(t)
	testkit.RequireProvidesOne[extension.SrvCfg](t, "server.configurators",
		IPExtractorModule(),
		fx.Supply(config.NewApplicationConfig(cfg), config.NewSecurityConfig(cfg)),
	)
}

func TestOpenAPIModule(t *testing.T) {
	t.Parallel()
	testkit.RequireProvidesOne[extension.UseMiddleware](t, "middlewares.use",
		OpenAPIModule(),
		fx.Supply(&openapi3.T{}),
		fx.Provide(func() echomw.Skipper { return nil }),
		fx.Provide(func() openapi3filter.AuthenticationFunc {
			return func(context.Context, *openapi3filter.AuthenticationInput) error { return nil }
		}),
	)
}

func TestURIModule(t *testing.T) {
	t.Parallel()
	testkit.RequireProvidesOne[extension.PreMiddleware](t, "middlewares.pre", URIModule())
}

func TestBodyLimitModule(t *testing.T) {
	t.Parallel()
	cfg := config.MockConfigForTest(t)
	testkit.RequireProvidesOne[extension.PreMiddleware](t, "middlewares.pre",
		BodyLimitModule(),
		fx.Supply(config.NewServerConfig(cfg)),
	)
}

func TestTimeoutModule(t *testing.T) {
	t.Parallel()
	cfg := config.MockConfigForTest(t)
	testkit.RequireProvidesOne[extension.PreMiddleware](t, "middlewares.pre",
		TimeoutModule(),
		fx.Supply(config.NewServerConfig(cfg)),
	)
}
