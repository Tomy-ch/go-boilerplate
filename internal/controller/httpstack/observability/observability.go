// Package observability は、オブザーバビリティに関する機能を提供します。
package observability

import (
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

// Middleware は、指定したサービス名で Echo 用の OTel ミドルウェアを返します。
func Middleware(serviceName string) echo.MiddlewareFunc {
	return otelecho.Middleware(serviceName)
}

// PassthroughMiddleware は、リクエストを次のハンドラへそのまま渡す素通しミドルウェアを返します。
func PassthroughMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return next
	}
}
