// Package forcejson は、特定のContent-Typeを強制的にapplication/jsonに設定します。
package forcejson

import (
	"strings"

	"github.com/labstack/echo/v4"
)

// Middleware は、Content-Type が未設定または text/html の場合に
// application/json へ強制するミドルウェアを返します。
func Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// ヘッダ確定(WriteHeader)直前に補正する。next 後にヘッダを変更しても、
			// ボディ書込で commit 済みのレスポンスには反映されないため Before フックで介入する。
			c.Response().Before(func() {
				ensureJSONContentType(c)
			})
			return next(c)
		}
	}
}

// ensureJSONContentType は、Content-Type が未設定または text/html の場合に application/json へ強制します。
func ensureJSONContentType(c echo.Context) {
	h := c.Response().Header()
	if shouldForceJSON(h.Get(echo.HeaderContentType)) {
		h.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
}

// shouldForceJSON は、Content-Type が未設定または text/html の場合に true を返します。
func shouldForceJSON(ct string) bool {
	return ct == "" || strings.HasPrefix(ct, echo.MIMETextHTML)
}
