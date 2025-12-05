package logging

import (
	"boilerplate-go/internal/controller/httpstack"
	"boilerplate-go/internal/logging"

	"go.uber.org/fx"
)

const priority = 8

// Module は、ロギング制御のミドルウェアを提供するfxモジュールを返します。
func Module() fx.Option {
	return fx.Module("mw.logging",
		fx.Provide(
			UseMiddleware,
		),
	)
}

// UseMiddleware は、OTelロギングミドルウェアを提供します。
func UseMiddleware(z logging.Logger, lf logging.LogFieldBuilder) httpstack.UseMiddlewareOut {
	return httpstack.UseMiddlewareOut{
		Middleware: httpstack.UseMiddleware{
			Name:       "logging",
			Priority:   priority,
			Middleware: Middleware(z, lf),
		},
	}
}
