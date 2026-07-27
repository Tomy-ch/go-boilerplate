// Package observability は、オブザーバビリティに関する機能を提供します。
package observability

import (
	echootel "github.com/labstack/echo-opentelemetry"
	"github.com/labstack/echo/v5"
)

// Middleware は、Echo 用の OTel ミドルウェアを返します。
// server.address / server.port は Request.Host から解決させるため Config.ServerName は空のままにします
// （ServerName はサーバの正式ホスト名であり、サービス名は OTel の Resource が持ちます）。
func Middleware() echo.MiddlewareFunc {
	return echootel.NewMiddlewareWithConfig(echootel.Config{})
}

// PassthroughMiddleware は、リクエストを次のハンドラへそのまま渡す素通しミドルウェアを返します。
func PassthroughMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return next
	}
}
