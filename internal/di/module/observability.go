package module

import (
	"go-boilerplate/internal/observability"

	"go.uber.org/fx"
)

// ObservabilityModule は、オブザーバビリティ（トレーシング）関連の依存関係を提供するfx.Moduleです。
func ObservabilityModule() fx.Option {
	return fx.Module("observability",
		fx.Provide(
			observability.TracerProvider,
			observability.NewTracerFactory,
		),
	)
}
