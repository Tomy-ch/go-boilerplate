// Package uri は、URI制御のミドルウェアを提供します。
package uri

import (
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// Middleware は、URIの末尾のスラッシュを削除するミドルウェアを返します。
func Middleware() echo.MiddlewareFunc {
	return middleware.RemoveTrailingSlash()
}
