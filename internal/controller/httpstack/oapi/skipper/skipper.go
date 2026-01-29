// Package skipper は、Oapiミドルウェアのスキッパー機能を提供します。
package skipper

import (
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

// New は、リクエストをスキップするためのSkipper関数を返します。
func New() echomw.Skipper {
	return func(c echo.Context) bool {
		p := c.Request().URL.Path
		switch p {
		case "/metrics":
			return true
		case "/health", "/healthz", "/ready", "/version":
			return true
		default:
			return false
		}
	}
}
