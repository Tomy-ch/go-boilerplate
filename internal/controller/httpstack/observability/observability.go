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
