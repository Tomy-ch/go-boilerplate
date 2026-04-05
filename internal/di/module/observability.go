package module

import (
	"go-boilerplate/internal/observability"

	"go.uber.org/fx"
)

func ObservabilityModule() fx.Option {
	return fx.Module("observability",
		fx.Provide(
			observability.TracerProvider,
			observability.NewTracerFactory,
		),
	)
}
