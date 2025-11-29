// Package observability は、オブザーバビリティに関する機能を提供します。
package observability

import (
	"context"

	"boilerplate-go/internal/config"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"go.uber.org/fx"
)

// Middleware は、Echo用のOTelミドルウェアを返します。
func Middleware(appCfg *config.ApplicationConfig) echo.MiddlewareFunc {
	return otelecho.Middleware(appCfg.AppName())
}

// TracerProvider は、OpenTelemetryのトレーサープロバイダーを初期化し、アプリケーションのライフサイクルに統合します。
func TracerProvider(lc fx.Lifecycle) trace.TracerProvider {
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(resource.Default()),
	)

	otel.SetTracerProvider(tp)

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return tp.Shutdown(ctx)
		},
	})

	return tp
}
