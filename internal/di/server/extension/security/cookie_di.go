package security

import (
	"go-boilerplate/internal/controller/httpstack/cookie"
	"go-boilerplate/internal/di/server/extension"

	"go.uber.org/fx"
)

const cookiePriority = 10

// CookieModule は、Cookie制御のミドルウェアを提供するfxモジュールを返します。
func CookieModule() fx.Option {
	return fx.Module("mw.secure_cookie",
		fx.Provide(
			CookieMiddleware,
		),
	)
}

// CookieMiddleware は、セキュアCookieのミドルウェアを提供します。
func CookieMiddleware(secCookie *cookie.SecurityCookie) extension.UseMiddlewareOut {
	return extension.UseMiddlewareOut{
		Middleware: extension.UseMiddleware{
			Name:       "secure_cookie",
			Priority:   cookiePriority,
			Middleware: cookie.Middleware(secCookie),
		},
	}
}
