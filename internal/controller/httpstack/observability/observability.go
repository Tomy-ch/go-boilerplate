// Package observability は、オブザーバビリティに関する機能を提供します。
package observability

import (
	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

// Middleware は、Echo用のOTelミドルウェアを返します。
func Middleware(appCfg *config.ApplicationConfig) echo.MiddlewareFunc {
	return otelecho.Middleware(appCfg.Name())
}
