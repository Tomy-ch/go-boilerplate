package requestid

import (
	"boilerplate-go/internal/controller/httpstack"

	"go.uber.org/fx"
)

const priority = 1

// Module は、リクエストID制御のミドルウェアを提供するfxモジュールを返します。
func Module() fx.Option {
	return fx.Module("mw.requestid",
		fx.Provide(
			UseMiddleware,
		),
	)
}

// UseMiddleware は、リクエストIDの生成ミドルウェアを提供します。
func UseMiddleware() httpstack.UseMiddlewareOut {
	return httpstack.UseMiddlewareOut{
		Middleware: httpstack.UseMiddleware{
			Priority:   priority,
			Middleware: Middleware(),
		},
	}
}
