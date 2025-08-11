// Package uricontrol は、URI制御のミドルウェアを提供します。
package uricontrol

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func Middleware() echo.MiddlewareFunc {
	return middleware.RemoveTrailingSlash()
}
