package outbound

import (
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/recovery"
	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/logging"

	"go.uber.org/fx"
)

const recoveryPriority = 3

// RecoveryModule は、リカバリ制御のミドルウェアを提供するfxモジュールを返します。
func RecoveryModule() fx.Option {
	return fx.Module("mw.recovery",
		fx.Provide(
			RecoveryMiddleware,
		),
	)
}

// RecoveryMiddleware は、リカバリ制御ミドルウェアを提供します。
func RecoveryMiddleware(
	logger logging.Logger, lf logging.LogFieldBuilder, appCfg *config.ApplicationConfig,
) extension.UseMiddlewareOut {
	return extension.UseMiddlewareOut{
		Middleware: extension.UseMiddleware{
			Name:       "recovery",
			Priority:   recoveryPriority,
			Middleware: recovery.Middleware(logger, lf, appCfg),
		},
	}
}
