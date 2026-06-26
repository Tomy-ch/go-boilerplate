// Package bodylimit は、リクエストボディのサイズ上限ミドルウェアを提供します。
package bodylimit

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Middleware は、リクエストボディを limitMB（MB, 10進・1MB=1,000,000 byte）で上限化する
// ミドルウェアを返します。上限超過時は echo が 413（Request Entity Too Large）を返します。
// limitMB が 0 以下の場合はパニックします。
//
// Pre ミドルウェアとして登録すること。Use 層の OpenAPI validator が requestBody を
// 読む前に上限を適用するために必要です。Use 以降に置くと validator が無制限ボディを
// 読み切り、上限がサイレントに無効化されます。
func Middleware(limitMB int) echo.MiddlewareFunc {
	if limitMB <= 0 {
		panic(fmt.Sprintf("bodylimit: limitMB must be positive, got %d", limitMB))
	}
	return middleware.BodyLimit(fmt.Sprintf("%dM", limitMB))
}
