// Package skipper は、Oapiミドルウェアのスキッパー機能を提供します。
package skipper

import (
	"boilerplate-go/internal/controller/httpstack/ops"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

// New は、リクエストをスキップするためのSkipper関数を返します。
func New() echomw.Skipper {
	return func(c echo.Context) bool {
		return ops.IsOpsPath(c.Request().URL.Path)
	}
}
