package observability

import (
	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/httpstack"

	"go.uber.org/fx"
)

const priority = 2

// Module は、可観測性のミドルウェアを提供するfxモジュールを返します。
func Module() fx.Option {
	return fx.Module("mw.observability",
		fx.Provide(
			UseMiddleware,
		),
	)
}

// UseMiddleware は、可観測性ミドルウェアを提供します。
func UseMiddleware(appCfg *config.ApplicationConfig) httpstack.UseMiddlewareOut {
	return httpstack.UseMiddlewareOut{
		Middleware: httpstack.UseMiddleware{
			Name:       "observability",
			Priority:   priority,
			Middleware: Middleware(appCfg),
		},
	}
}
