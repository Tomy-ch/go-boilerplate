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

// GetRequestIDFromResponse は、レスポンスのEchoコンテキストからリクエストIDを取得します。
func GetRequestIDFromResponse(
	c echo.Context,
) string {
	return c.Request().Header.Get(echo.HeaderXRequestID)
}
