package core

import (
	"go-boilerplate/internal/controller/httpstack/cookie"

	"go.uber.org/fx"
)

// SecurityCookieModule は、セキュリティCookieのコア機能部分を提供するfxモジュールを返します。
func SecurityCookieModule() fx.Option {
	return fx.Module("core.security_cookie",
		fx.Provide(
			cookie.NewSecurityCookie,
		),
	)
}
