// Package bodylimit は、リクエストボディのサイズ上限ミドルウェアを提供します。
package bodylimit

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Middleware は、リクエストボディを limitMB（MB, 10進・1MB=1,000,000 byte）で上限化する
// ミドルウェアを返します。上限超過時は echo が 413（Request Entity Too Large）を返します。
//
// echo 標準の middleware.BodyLimit を薄くラップします。BodyLimit は reader をラップするだけで
// ルーティングに非依存なため、Pre ミドルウェアとして登録すれば OpenAPI validator（Use）が
// requestBody を読み切る前に確実に上限を適用できます（M2）。
func Middleware(limitMB int) echo.MiddlewareFunc {
	return middleware.BodyLimit(fmt.Sprintf("%dM", limitMB))
}
