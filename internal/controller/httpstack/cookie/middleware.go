package cookie

import (
	"github.com/labstack/echo/v4"
)

// Middleware は Echo 用 middleware です。
func Middleware(cfg *SecurityCookie) echo.MiddlewareFunc {
	return secureCookieMiddleware(cfg)
}

// secureCookieMiddleware は、SecurityCookie 設定に基づいてクッキーを書き換える Echo 用の middleware を生成します。
func secureCookieMiddleware(cfg *SecurityCookie) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			res := c.Response()
			orig := res.Writer

			w := newCookieRewriteWriter(orig, cfg)
			res.Writer = w
			defer func() { res.Writer = orig }()

			return next(c)
		}
	}
}
