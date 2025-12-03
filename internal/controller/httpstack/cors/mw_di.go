package cors

import (
	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/httpstack"

	"go.uber.org/fx"
)

const priority = 4

// Module は、CORS制御のミドルウェアを提供するfxモジュールを返します。
func Module() fx.Option {
	return fx.Module("mw.cors",
		fx.Provide(
			UseMiddleware,
		),
	)
}

// UseMiddleware は、CORS制御のミドルウェアを生成します。
func UseMiddleware(secCfg *config.SecurityConfig) httpstack.UseMiddlewareOut {
	return httpstack.UseMiddlewareOut{
		Middleware: httpstack.UseMiddleware{
			Name:       "cors",
			Priority:   priority,
			Middleware: Middleware(secCfg),
		},
	}
}
