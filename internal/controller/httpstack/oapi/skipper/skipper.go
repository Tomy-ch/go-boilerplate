// Package skipper は、Oapiミドルウェアのスキッパー機能を提供します。
package skipper

import (
	"go-boilerplate/internal/controller/httpstack/ops"

	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"
)

// New は、運用系エンドポイント（/health, /metrics 等）へのリクエストを OpenAPI バリデーションからスキップする Skipper 関数を返します。
func New() echomw.Skipper {
	return func(c *echo.Context) bool {
		return ops.IsOpsPath(c.Request().URL.Path)
	}
}
