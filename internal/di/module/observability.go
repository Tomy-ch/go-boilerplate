package module

import (
	"go-boilerplate/internal/di/server/hook"
	"go-boilerplate/internal/observability"

	"go.uber.org/fx"
)

// ObservabilityModule は、オブザーバビリティ（トレーシング / メトリクス）関連の依存関係を提供するfx.Moduleです。
func ObservabilityModule() fx.Option {
	return fx.Module("observability",
		fx.Provide(
			observability.NewResource,
			observability.NewTracerProvider,
			observability.NewMeterProvider,
			observability.ProvideTracerProvider,
			observability.NewProviderShutdowner,
			observability.NewTracerFactory,
		),
		fx.Invoke(hook.RegisterObservabilityShutdownHooks),
	)
}
