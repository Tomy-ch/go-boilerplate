// Package logging は、Echoフレームワークのミドルウェアとしてリクエストのログの出力を提供します。
package logging

import (
	"net/http"
	"time"

	"boilerplate-go/internal/appconfig"
	"boilerplate-go/internal/controller/ctxhelper"
	"boilerplate-go/internal/domain/expectederrors"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Middleware は、Echoフレームワークのミドルウェアで、リクエストのログを出力します。
func Middleware(
	logger *zap.Logger,
	cfg *appconfig.Config,
) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			latency := time.Since(start)

			fields := buildFields(c, latency, err)
			logRequest(c, logger, fields)
			logErrorInDev(logger, cfg, err)
			return err
		}
	}
}

// buildFields は、リクエストの情報を含むzap.Fieldのスライスを生成します。
func buildFields(c echo.Context, latency time.Duration, err error) []zap.Field {
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
		zap.String("trace_id", spanCtx.TraceID().String()),
		zap.String("span_id", spanCtx.SpanID().String()),
	}

	if err != nil {
		fields = append(fields,
			zap.Error(err),
		)
	}

	return fields
}

// logRequest は、リクエストのログを出力します。
// ステータスコードに応じて、エラーログ、警告ログ、または情報ログを出力します。
func logRequest(c echo.Context, logger *zap.Logger, fields []zap.Field) {
	status := c.Response().Status

	switch {
	case status >= MinStatusError:
		logger.Error("server error", fields...)
	case status == http.StatusNotFound && isExpectedNotFound(c):
		logger.Info("expected not found", fields...)
	case status >= MinStatusWarn:
		logger.Warn("client error", fields...)
	default:
		logger.Info("request handled", fields...)
	}
}

// logErrorInDev は、開発環境でのみのログを出力します。
func logErrorInDev(logger *zap.Logger, cfg *appconfig.Config, err error) {
	if cfg.IsAppDevelopmentMode() && err != nil {
		logger.Error(err.Error())
	}
}

// isExpectedNotFound は、リクエストが期待される404 Not Foundエラーであるかどうかを判定します。
func isExpectedNotFound(c echo.Context) bool {
	val, ok := ctxhelper.GetNotFoundFromEcho(c)
	return ok && expectederrors.IsDefinedNotFoundCause(val)
}
