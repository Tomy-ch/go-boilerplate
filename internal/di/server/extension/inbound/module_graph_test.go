package inbound

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di/server/extension"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	echomw "github.com/labstack/echo/v4/middleware"
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

func TestBinderModule_ProvidesServeConfig(t *testing.T) {
	t.Parallel()
	requireProvidesOne[extension.SrvCfg](t, "server.configurators", BinderModule())
}

func TestIPExtractorModule_ProvidesServeConfig(t *testing.T) {
	t.Parallel()
	cfg := config.MockConfigForTest(t)
	requireProvidesOne[extension.SrvCfg](t, "server.configurators",
		IPExtractorModule(),
		fx.Supply(config.NewApplicationConfig(cfg), config.NewSecurityConfig(cfg)),
	)
}

func TestOpenAPIModule_ProvidesUseMiddleware(t *testing.T) {
	t.Parallel()
	requireProvidesOne[extension.UseMiddleware](t, "middlewares.use",
		OpenAPIModule(),
		fx.Supply(&openapi3.T{}),
		fx.Provide(func() echomw.Skipper { return nil }),
		fx.Provide(func() openapi3filter.AuthenticationFunc {
			return func(context.Context, *openapi3filter.AuthenticationInput) error { return nil }
		}),
	)
}

func TestURIModule_ProvidesPreMiddleware(t *testing.T) {
	t.Parallel()
	requireProvidesOne[extension.PreMiddleware](t, "middlewares.pre", URIModule())
}
