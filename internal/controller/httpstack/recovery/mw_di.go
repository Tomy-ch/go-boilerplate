package recovery

import (
	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/httpstack"
	"boilerplate-go/internal/logging"

	"go.uber.org/fx"
)

const priority = 3

// Module は、リカバリ制御のミドルウェアを提供するfxモジュールを返します。
func Module() fx.Option {
	return fx.Module("mw.recovery",
		fx.Provide(
			UseMiddleware,
		),
	)
}

// UseMiddleware は、リカバリ制御ミドルウェアを提供します。
func UseMiddleware(logger logging.Logger, lf logging.LogFieldBuilder, appCfg *config.ApplicationConfig) httpstack.UseMiddlewareOut {
	return httpstack.UseMiddlewareOut{
		Middleware: httpstack.UseMiddleware{
			Name:       "recovery",
			Priority:   priority,
			Middleware: Middleware(logger, lf, appCfg),
		},
	}
}
