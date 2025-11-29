package logging

import (
	"boilerplate-go/internal/controller/httpstack"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

const priority = 2

// Module は、ロギング制御のミドルウェアを提供するfxモジュールを返します。
func Module() fx.Option {
	return fx.Module("mw.logging",
		fx.Provide(
			UseMiddleware,
		),
	)
}

// UseMiddleware は、OTelロギングミドルウェアを提供します。
func UseMiddleware(z *zap.Logger) httpstack.UseMiddlewareOut {
	return httpstack.UseMiddlewareOut{
		Middleware: httpstack.UseMiddleware{
			Priority:   priority,
			Middleware: Middleware(z),
		},
	}
}
