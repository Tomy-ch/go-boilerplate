// Package security は、セキュリティ制御系のミドルウェアをDIコンテナに提供します。
package security

import (
	"go-boilerplate/internal/config"
	httpsecurity "go-boilerplate/internal/controller/httpstack/security"
	"go-boilerplate/internal/di/server/extension"

	"go.uber.org/fx"
)

const securityPriority = 5

// Module は、セキュリティ制御のミドルウェアを提供するfxモジュールを返します。
func Module() fx.Option {
	return fx.Module("mw.security",
		fx.Provide(
			Middleware,
		),
	)
}

// Middleware は、セキュリティミドルウェアを提供します。
func Middleware(secCfg *config.SecurityConfig) extension.UseMiddlewareOut {
	return extension.UseMiddlewareOut{
		Middleware: extension.UseMiddleware{
			Name:       "security",
			Priority:   securityPriority,
			Middleware: httpsecurity.Middleware(secCfg),
		},
	}
}
