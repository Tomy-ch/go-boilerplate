package module

import (
	"go-boilerplate/internal/di/server/hook"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/observability/metrics/buildinfo"

	"go.uber.org/fx"
)

// ObservabilityModule は、オブザーバビリティ（トレーシング / メトリクス / ロギング）関連の依存関係を提供するfx.Moduleです。
func ObservabilityModule() fx.Option {
	return fx.Module("observability",
		fx.Provide(
			observability.NewResource,
			observability.NewTracerProvider,
			observability.NewMeterProvider,
			observability.NewLoggerProvider,
			observability.ProvideTracerProvider,
			observability.ProvideMeterProvider,
			observability.NewProviderShutdowner,
			observability.NewTracerFactory,
			observability.NewLogCore,
			observability.NewTraceExtractor,
			observability.NewPgxTracer,
			observability.NewTextMapPropagator,
			observability.NewHTTPClientTransport,
			observability.NewHTTPClientMetrics,
			buildinfo.NewCollector,
		),
		fx.Invoke(
			hook.RegisterObservabilityShutdownHooks,
			buildinfo.Register,
		),
	)
}
