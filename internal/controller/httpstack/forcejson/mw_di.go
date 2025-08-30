package forcejson

import "go.uber.org/fx"

// Module は、JOINの強制制御のミドルウェアを提供するfxモジュールを返します。
func Module() fx.Option {
	return fx.Module("mw.forcejson",
		fx.Provide(
			fx.Annotate(
				Middleware,
				fx.ResultTags(`group:"middlewares.use"`),
			),
		),
	)
}
