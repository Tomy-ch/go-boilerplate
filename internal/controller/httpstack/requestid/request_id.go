// Package requestid は、リクエストIDミドルウェアをラップするためのパッケージです。
package requestid

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Middleware は、リクエストIDを生成するミドルウェアを返します。
func Middleware() echo.MiddlewareFunc {
	return middleware.RequestID()
}

// GetRequestIDFromResponse は、レスポンスヘッダ X-Request-ID からリクエストIDを取得します（Middleware が設定した値を読み出す）。
func GetRequestIDFromResponse(
	c echo.Context,
) string {
	return c.Response().Header().Get(echo.HeaderXRequestID)
}
