// Package cookie は、クッキーの使用を制御するミドルウェアを提供します。
package cookie

import (
	"github.com/labstack/echo/v4"
)

// Middleware は、クッキーの使用を制御するミドルウェアを返します。
func Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Request().Header.Del(echo.HeaderCookie)
			c.Response().Header().Del(echo.HeaderSetCookie)
			return next(c)
		}
	}
}
