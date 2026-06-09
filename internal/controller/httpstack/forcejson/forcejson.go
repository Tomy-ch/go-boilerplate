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
			// next 後では commit 済みで反映されないため、WriteHeader 直前(Before)に補正する。
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
