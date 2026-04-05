// Package instrumentation は、サーバーの監視・計測に関するDIモジュールを提供します。
package instrumentation

import (
	hslogger "go-boilerplate/internal/controller/httpstack/logging"
	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/logging"

	"go.uber.org/fx"
)

const loggingPriority = 8

// LoggingModule は、ロギング制御のミドルウェアを提供するfxモジュールを返します。
func LoggingModule() fx.Option {
	return fx.Module("mw.logging",
		fx.Provide(
			LoggingMiddleware,
		),
	)
}

// LoggingMiddleware は、OTelロギングミドルウェアを提供します。
func LoggingMiddleware(z logging.Logger, lf logging.LogFieldBuilder) extension.UseMiddlewareOut {
	return extension.UseMiddlewareOut{
		Middleware: extension.UseMiddleware{
			Name:       "logging",
			Priority:   loggingPriority,
			Middleware: hslogger.Middleware(z, lf),
		},
	}
}
