// Package uri は、URI制御のミドルウェアを提供します。
package uri

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Middleware は、URIの末尾のスラッシュを削除するミドルウェアを返します。
func Middleware() echo.MiddlewareFunc {
	return middleware.RemoveTrailingSlash()
}
