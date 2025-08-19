package di

import "go.uber.org/fx"

// Module は、コントローラー層の依存関係を提供するfx.Moduleです。
func Module() fx.Option {
	return fx.Module("controller",
		fx.Provide(),
	)
}
