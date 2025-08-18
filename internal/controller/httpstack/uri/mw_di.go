package uri

import (
	"go.uber.org/fx"
)

// Module は、URI制御のミドルウェアを提供するfxモジュールを返します。
func Module() fx.Option {
	return fx.Module("mw.uri",
		fx.Provide(
			fx.Annotate(
				Middleware,
				fx.ResultTags(`group:"middlewares.pre"`),
			),
		),
	)
}
