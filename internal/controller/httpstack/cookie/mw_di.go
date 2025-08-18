package cookie

import "go.uber.org/fx"

// Module は、Cookie制御のミドルウェアを提供するfxモジュールを返します。
func Module() fx.Option {
	return fx.Module("mw.cookie",
		fx.Provide(
			fx.Annotate(
				Middleware,
				fx.ResultTags(`group:"middlewares.use"`),
			),
		),
	)
}
