package observability

import (
	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/httpstack"
	"boilerplate-go/internal/observability"

	"go.uber.org/fx"
)

const priority = 1

// Module は、可観測性のミドルウェアを提供するfxモジュールを返します。
func Module() fx.Option {
	return fx.Module("mw.observability",
		fx.Provide(
			UseMiddleware,
		),
	)
}

// UseMiddleware は、可観測性ミドルウェアを提供します。
func UseMiddleware(appCfg *config.ApplicationConfig) httpstack.UseMiddlewareOut {
	return httpstack.UseMiddlewareOut{
		Middleware: httpstack.UseMiddleware{
			Priority:   priority,
			Middleware: Middleware(appCfg),
		},
	}
}

// PrimitiveModule は、可観測性ミドルウェアの初期化とトレーサープロバイダーの提供を行うfxモジュールを返します。
func PrimitiveModule() fx.Option {
	return fx.Module("observability.tracerProvider",
		fx.Provide(
			TracerProvider,
		),
	)
}

// CoreModule は、可観測性のトレーサーコアを提供するfxモジュールを返します。
func CoreModule() fx.Option {
	return fx.Module("observability.tracerFactory",
		fx.Provide(
			observability.NewTracerFactory,
		),
	)
}
