// Package forcejson は、特定のContent-Typeを強制的にapplication/jsonに設定します。
package forcejson

import (
	"strings"

	"github.com/labstack/echo/v4"
)

const (
	charsetUTF8 = "charset=UTF-8"
)

// Middleware は、特定のContent-Typeを強制的にapplication/jsonに設定します。
func Middleware() echo.MiddlewareFunc {
	return forceJSONContentTypeMiddleware()
}

// forceJSONContentTypeMiddleware は、レスポンスのContent-Typeを強制的にapplication/jsonに設定するミドルウェアを返します。
func forceJSONContentTypeMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if err := next(c); err != nil {
				return err
			}

			ensureJSONContentType(c)

			return nil
		}
	}
}

// ensureJSONContentType は、レスポンスのContent-Typeがブラックリストに登録されている場合に、application/jsonに強制的に設定します。
func ensureJSONContentType(c echo.Context) {
	h := c.Response().Header()
	ct := h.Get(echo.HeaderContentType)

	if isBlacklistedContentType(ct) {
		h.Set(echo.HeaderContentType, jsonContentTypeWithCharset())
	}
}

// jsonContentTypeWithCharset は、application/json;charset=UTF-8 を返します。
func jsonContentTypeWithCharset() string {
	return echo.MIMEApplicationJSON + "; " + charsetUTF8
}

// isBlacklistedContentType は、特定のContent-Typeがブラックリストに登録されているかどうかを判定します。
func isBlacklistedContentType(ct string) bool {
	return ct == "" || strings.HasPrefix(ct, echo.MIMETextHTML)
}
