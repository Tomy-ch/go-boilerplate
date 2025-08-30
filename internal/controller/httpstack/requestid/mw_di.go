package requestid

import "go.uber.org/fx"

// Module は、リクエストID制御のミドルウェアを提供するfxモジュールを返します。
func Module() fx.Option {
	return fx.Module("mw.requestid",
		fx.Provide(
			fx.Annotate(
				Middleware,
				fx.ResultTags(`group:"middlewares.use"`),
			),
		),
	)
}
