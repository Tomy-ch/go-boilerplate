// Package instrumentation は、サーバーの監視・計測に関するDIモジュールを提供します。
package instrumentation

import (
	hslogger "go-boilerplate/internal/controller/httpstack/logging"
	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/logging"

	"go.uber.org/fx"
)

// loggingPriority は、ロギングミドルウェアの適用順序です。
// 順序設計の根拠は README「Priority Order」を参照してください。
const loggingPriority = 9

// LoggingModule は、ロギング制御のミドルウェアを提供するfxモジュールを返します。
func LoggingModule() fx.Option {
	return fx.Module("mw.logging",
		fx.Provide(
			LoggingMiddleware,
		),
	)
}

// LoggingMiddleware は、HTTPリクエスト／レスポンスのアクセスログを出力するミドルウェアを提供します（OTel トレースコンテキストの TraceID・SpanID を含む）。
func LoggingMiddleware(z logging.Logger, lf logging.LogFieldBuilder) extension.UseMiddlewareOut {
	return extension.UseMiddlewareOut{
		Middleware: extension.UseMiddleware{
			Name:       "logging",
			Priority:   loggingPriority,
			Middleware: hslogger.Middleware(z, lf),
		},
	}
}
