package cookie

import (
	"go-boilerplate/internal/controller/server"

	"github.com/labstack/echo/v5"
)

// Middleware は、SecurityCookie 設定に従い Set-Cookie ヘッダのセキュリティ属性（Secure / HttpOnly / SameSite / Path / Domain など）を上書きする Echo ミドルウェアを返します。
func Middleware(cfg *SecurityCookie) echo.MiddlewareFunc {
	return secureCookieMiddleware(cfg)
}

// secureCookieMiddleware は、SecurityCookie 設定に基づいてクッキーを書き換える Echo 用の middleware を生成します。
// レスポンスを取り出せない場合は書き換えず素通しします。
func secureCookieMiddleware(cfg *SecurityCookie) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			res := server.ResponseOf(c)
			if res == nil {
				return next(c)
			}

			// next 後に ResponseWriter を復元しない（エラー経路の Set-Cookie も書き換え対象にするため。後始末は echo の Context Reset が担う）。
			res.ResponseWriter = newCookieRewriteWriter(res.ResponseWriter, cfg)

			return next(c)
		}
	}
}
