package security

import (
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/cookie"
	"go-boilerplate/internal/controller/httpstack/ratelimit"
	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/di/server/extension/testkit"

	"go.uber.org/fx"
)

func TestCookieModule_ProvidesUseMiddleware(t *testing.T) {
	t.Parallel()
	testkit.RequireProvidesOne[extension.UseMiddleware](t, "middlewares.use",
		CookieModule(),
		fx.Provide(cookie.NewSecurityCookie),
		fx.Supply(config.NewSecureCookieConfig(config.MockConfigForTest(t))),
	)
}

func TestCORSModule_ProvidesUseMiddleware(t *testing.T) {
	t.Parallel()
	testkit.RequireProvidesOne[extension.UseMiddleware](t, "middlewares.use",
		CORSModule(),
		fx.Supply(config.NewSecurityConfig(config.MockConfigForTest(t))),
	)
}

func TestRateLimitModule_ProvidesUseMiddleware(t *testing.T) {
	t.Parallel()
	testkit.RequireProvidesOne[extension.UseMiddleware](t, "middlewares.use",
		RateLimitModule(),
		fx.Provide(ratelimit.NewIPRateLimiter),
		fx.Supply(config.NewIPRateLimitConfig(config.MockConfigForTest(t))),
	)
}

func TestSecurityModule_ProvidesUseMiddleware(t *testing.T) {
	t.Parallel()
	testkit.RequireProvidesOne[extension.UseMiddleware](t, "middlewares.use",
		Module(),
		fx.Supply(config.NewSecurityConfig(config.MockConfigForTest(t))),
	)
}
