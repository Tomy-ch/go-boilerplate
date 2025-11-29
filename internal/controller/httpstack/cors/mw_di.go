package cors

import (
	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/httpstack"

	"go.uber.org/fx"
)

const priority = 1

// Module は、CORS制御のミドルウェアを提供するfxモジュールを返します。
func Module() fx.Option {
	return fx.Module("mw.cors",
		fx.Provide(
			UseMiddleware,
		),
	)
}

// UseMiddleware は、可観測性ミドルウェアを提供します。
func UseMiddleware(secCfg *config.SecurityConfig) httpstack.UseMiddlewareOut {
	return httpstack.UseMiddlewareOut{
		Middleware: httpstack.UseMiddleware{
			Priority:   priority,
			Middleware: Middleware(secCfg),
		},
	}
}
