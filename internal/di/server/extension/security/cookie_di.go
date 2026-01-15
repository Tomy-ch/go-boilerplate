package security

import (
	"boilerplate-go/internal/controller/httpstack/cookie"
	"boilerplate-go/internal/di/server/extension"

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

// CookieMiddleware は、可観測性ミドルウェアを提供します。
func CookieMiddleware(secCookie *cookie.SecurityCookie) extension.UseMiddlewareOut {
	return extension.UseMiddlewareOut{
		Middleware: extension.UseMiddleware{
			Name:       "secure_cookie",
			Priority:   cookiePriority,
			Middleware: cookie.Middleware(secCookie),
		},
	}
}
