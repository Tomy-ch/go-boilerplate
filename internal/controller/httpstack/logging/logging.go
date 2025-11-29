// Package logging は、Echoフレームワークのミドルウェアとしてリクエストのログの出力を提供します。
package logging

import (
	"time"

	"boilerplate-go/internal/controller/httpstack/requestid"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Middleware は、Echoフレームワークのミドルウェアで、リクエストのログを出力します。
func Middleware(logger *zap.Logger) echo.MiddlewareFunc {
	return requestLogMiddleware(logger)
}

// requestLogMiddleware は、リクエストのログを出力するミドルウェアを返します。
func requestLogMiddleware(logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			c.Response().After(func() {
				latency := time.Since(start)

				fields := buildRequestLogFields(c, latency)
				writeRequestLog(c, logger, fields)
			})

			return next(c)
		}
	}
}

// buildRequestLogFields は、リクエストの情報を含むzap.Fieldのスライスを生成します。
func buildRequestLogFields(c echo.Context, latency time.Duration) []zap.Field {
	req := c.Request()
	res := c.Response()
	status := res.Status
	spanCtx := trace.SpanFromContext(req.Context()).SpanContext()

	fields := []zap.Field{
		zap.String("method", req.Method),
		zap.String("uri", req.RequestURI),
		zap.Int("status", status),
		zap.Duration("latency", latency),
		zap.String("remote_ip", c.RealIP()),
		zap.String("request_id", requestid.GetRequestIDFromResponse(c)),
		zap.String("trace_id", spanCtx.TraceID().String()),
		zap.String("span_id", spanCtx.SpanID().String()),
	}

	return fields
}

// writeRequestLog は、リクエストのログを出力します。
// ステータスコードに応じて、エラーログ、警告ログ、または情報ログを出力します。
func writeRequestLog(c echo.Context, logger *zap.Logger, fields []zap.Field) {
	status := c.Response().Status
	switch {
	case status >= MinStatusError:
		logger.Error("server error", fields...)
	case status >= MinStatusWarn:
		logger.Warn("client error", fields...)
	default:
		logger.Info("request handled", fields...)
	}
}
