package uri

import (
	"boilerplate-go/internal/controller/httpstack"

	"go.uber.org/fx"
)

const priority = 6

// Module は、URI制御のミドルウェアを提供するfxモジュールを返します。
func Module() fx.Option {
	return fx.Module("mw.uri",
		fx.Provide(
			PreMiddleware,
		),
	)
}

// PreMiddleware は、バリデーションミドルウェアを提供します。
func PreMiddleware() httpstack.PreMiddlewareOut {
	return httpstack.PreMiddlewareOut{
		Middleware: httpstack.PreMiddleware{
			Name:       "uri",
			Priority:   priority,
			Middleware: Middleware(),
		},
	}
}
