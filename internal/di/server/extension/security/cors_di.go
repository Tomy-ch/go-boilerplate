package security

import (
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/cors"
	"go-boilerplate/internal/di/server/extension"

	"go.uber.org/fx"
)

const corsPriority = 4

// CORSModule は、CORS制御のミドルウェアを提供するfxモジュールを返します。
func CORSModule() fx.Option {
	return fx.Module("mw.cors",
		fx.Provide(
			CORSMiddleware,
		),
	)
}

// CORSMiddleware は、CORS制御のミドルウェアを生成します。
func CORSMiddleware(secCfg *config.SecurityConfig) extension.UseMiddlewareOut {
	return extension.UseMiddlewareOut{
		Middleware: extension.UseMiddleware{
			Name:       "cors",
			Priority:   corsPriority,
			Middleware: cors.Middleware(secCfg),
		},
	}
}
