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
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			if err != nil {
				return err
			}
			ct := c.Response().Header().Get(echo.HeaderContentType)
			if isBlacklistedContentType(ct) {
				c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON+"; "+charsetUTF8)
			}
			return nil
		}
	}
}

// isBlacklistedContentType は、特定のContent-Typeがブラックリストに登録されているかどうかを判定します。
func isBlacklistedContentType(ct string) bool {
	return ct == "" || strings.HasPrefix(ct, echo.MIMETextHTML)
}
