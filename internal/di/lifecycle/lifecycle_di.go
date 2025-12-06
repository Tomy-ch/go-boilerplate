package lifecycle

import "go.uber.org/fx"

// Module は、LifecycleRegistrarを提供するfx.Moduleを返します。
func Module() fx.Option {
	return fx.Module("lifecycle.registrar",
		fx.Provide(
			NewLifecycleRegistrar,
		),
	)
}
