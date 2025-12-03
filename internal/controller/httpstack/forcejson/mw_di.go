package forcejson

import (
	"boilerplate-go/internal/controller/httpstack"

	"go.uber.org/fx"
)

const priority = 7

// Module は、JOINの強制制御のミドルウェアを提供するfxモジュールを返します。
func Module() fx.Option {
	return fx.Module("mw.forcejson",
		fx.Provide(
			UseMiddleware,
		),
	)
}

// UseMiddleware は、JOINの強制制御ミドルウェアを提供します。
func UseMiddleware() httpstack.UseMiddlewareOut {
	return httpstack.UseMiddlewareOut{
		Middleware: httpstack.UseMiddleware{
			Name:       "forcejson",
			Priority:   priority,
			Middleware: Middleware(),
		},
	}
}
