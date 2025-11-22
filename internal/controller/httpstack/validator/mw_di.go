package validator

import "go.uber.org/fx"

// Module は、OpenAPIバリデーションのミドルウェアを提供するfxモジュールを返します。
func Module() fx.Option {
	return fx.Module("mw.validator",
		fx.Provide(
			fx.Annotate(
				Middleware,
				fx.ResultTags(`group:"middlewares.use"`),
			),
		),
	)
}

// CoreModule は、ルーティング時に自動で解決されるバリデーションのコア機能部分を提供するfxモジュールを返します。
func CoreModule() fx.Option {
	return fx.Module("validator.core",
		fx.Provide(
			GetValidator,
		),
	)
}
