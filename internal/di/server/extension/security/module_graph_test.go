package security

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/cookie"
	"go-boilerplate/internal/controller/httpstack/ratelimit"
	"go-boilerplate/internal/di/server/extension"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

// requireProvidesOne は、与えたモジュール群が指定 group に要素をちょうど1件 provide することを検証する。
// fx.NopLogger は構成時のログ出力を抑えるだけで、検証結果には影響しない。
func requireProvidesOne[T any](t *testing.T, group string, opts ...fx.Option) {
	t.Helper()

	var got []T
	app := fx.New(append(opts,
		fx.Populate(fx.Annotate(&got, fx.ParamTags(`group:"`+group+`"`))),
		fx.NopLogger,
	)...)

	require.NoError(t, app.Start(context.Background()))
	defer func() { require.NoError(t, app.Stop(context.Background())) }()

	require.Len(t, got, 1)
}

func TestCookieModule_ProvidesUseMiddleware(t *testing.T) {
	t.Parallel()
	requireProvidesOne[extension.UseMiddleware](t, "middlewares.use",
		CookieModule(),
		fx.Provide(cookie.NewSecurityCookie),
		fx.Supply(config.NewSecureCookieConfig(config.MockConfigForTest(t))),
	)
}

func TestCORSModule_ProvidesUseMiddleware(t *testing.T) {
	t.Parallel()
	requireProvidesOne[extension.UseMiddleware](t, "middlewares.use",
		CORSModule(),
		fx.Supply(config.NewSecurityConfig(config.MockConfigForTest(t))),
	)
}

func TestRateLimitModule_ProvidesUseMiddleware(t *testing.T) {
	t.Parallel()
	requireProvidesOne[extension.UseMiddleware](t, "middlewares.use",
		RateLimitModule(),
		fx.Provide(ratelimit.NewIPRateLimiter),
		fx.Supply(config.NewIPRateLimitConfig(config.MockConfigForTest(t))),
	)
}

func TestSecurityModule_ProvidesUseMiddleware(t *testing.T) {
	t.Parallel()
	requireProvidesOne[extension.UseMiddleware](t, "middlewares.use",
		Module(),
		fx.Supply(config.NewSecurityConfig(config.MockConfigForTest(t))),
	)
}
