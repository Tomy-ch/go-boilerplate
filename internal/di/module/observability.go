package module

import (
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/observability/metrics/buildinfo"

	"go.uber.org/fx"
)

func ObservabilityModule() fx.Option {
	return fx.Module("observability",
		fx.Provide(
			observability.TracerProvider,
			observability.NewTracerFactory,
			buildinfo.NewCollector,
		),
		fx.Invoke(
			buildinfo.Register,
		),
	)
}
