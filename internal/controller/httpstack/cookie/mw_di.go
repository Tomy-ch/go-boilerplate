package cookie

import (
	"boilerplate-go/internal/controller/httpstack"

	"go.uber.org/fx"
)

const priority = 10

// Module は、Cookie制御のミドルウェアを提供するfxモジュールを返します。
func Module() fx.Option {
	return fx.Module("mw.cookie",
		fx.Provide(
			UseMiddleware,
		),
	)
}

// UseMiddleware は、可観測性ミドルウェアを提供します。
func UseMiddleware() httpstack.UseMiddlewareOut {
	return httpstack.UseMiddlewareOut{
		Middleware: httpstack.UseMiddleware{
			Name:       "cookie",
			Priority:   priority,
			Middleware: Middleware(),
		},
	}
}
