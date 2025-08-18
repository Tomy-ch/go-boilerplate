package cors

import "go.uber.org/fx"

// Module は、CORS制御のミドルウェアを提供するfxモジュールを返します。
func Module() fx.Option {
	return fx.Module("mw.cors",
		fx.Provide(
			fx.Annotate(
				Middleware,
				fx.ResultTags(`group:"middlewares.use"`),
			),
		),
	)
}
