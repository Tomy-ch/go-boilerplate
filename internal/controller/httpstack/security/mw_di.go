package security

import (
	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/httpstack"

	"go.uber.org/fx"
)

const priority = 5

// Module は、セキュリティ制御のミドルウェアを提供するfxモジュールを返します。
func Module() fx.Option {
	return fx.Module("mw.security",
		fx.Provide(
			UseMiddleware,
		),
	)
}

// UseMiddleware は、セキュリティミドルウェアを提供します。
func UseMiddleware(appCfg *config.ApplicationConfig) httpstack.UseMiddlewareOut {
	return httpstack.UseMiddlewareOut{
		Middleware: httpstack.UseMiddleware{
			Name:       "security",
			Priority:   priority,
			Middleware: Middleware(appCfg),
		},
	}
}
