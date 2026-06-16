package module

import (
	"go-boilerplate/internal/observability"

	"go.uber.org/fx"
)

// ObservabilityModule は、オブザーバビリティ（トレーシング / メトリクス）関連の依存関係を提供するfx.Moduleです。
func ObservabilityModule() fx.Option {
	return fx.Module("observability",
		fx.Provide(
			observability.NewResource,
			observability.TracerProvider,
			observability.MeterProvider,
			observability.NewTracerFactory,
		),
		// MeterProvider は他に依存元が無いため、明示的に構築させる。
		fx.Invoke(observability.InvokeMeterProvider),
	)
}
