package shutdowner

import "go.uber.org/fx"

// Module は、Shutdownerを提供するfx.Moduleを返します。
func Module() fx.Option {
	return fx.Module("shutdowner",
		fx.Provide(
			NewShutdowner,
		),
	)
}
