package instrumentation

import (
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/observability"
	"go-boilerplate/internal/di/server/extension"

	"go.uber.org/fx"
)

const observabilityPriority = 2

// ObservabilityModule は、可観測性のミドルウェアを提供するfxモジュールを返します。
func ObservabilityModule() fx.Option {
	return fx.Module("mw.observability",
		fx.Provide(
			ObservabilityMiddleware,
		),
	)
}

// ObservabilityMiddleware は、可観測性ミドルウェアを提供します。
func ObservabilityMiddleware(appCfg *config.ApplicationConfig) extension.UseMiddlewareOut {
	return extension.UseMiddlewareOut{
		Middleware: extension.UseMiddleware{
			Name:       "observability",
			Priority:   observabilityPriority,
			Middleware: observability.Middleware(appCfg.Name()),
		},
	}
}
