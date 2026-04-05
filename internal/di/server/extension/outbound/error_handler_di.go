// Package outbound は、サーバーの応答や出力時の拡張機能に関するDIモジュールを提供します。
package outbound

import (
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/errorhandler"
	"go-boilerplate/internal/di/server/extension"
	"go-boilerplate/internal/logging"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
)

// ErrorHandlerModule は、Error Handler モジュールを提供します。
func ErrorHandlerModule() fx.Option {
	return fx.Module("server.errorhandler",
		fx.Provide(
			provideErrorHandlerServeConfig,
		),
	)
}

// provideErrorHandlerServeConfig は、Error Handler のサーバー設定を提供します。
func provideErrorHandlerServeConfig(
	log logging.Logger, lf logging.LogFieldBuilder, obsCfg *config.ObservabilityConfig,
) extension.ServeCfgOut {
	return extension.ServeCfgOut{
		SrvCfg: func(e *echo.Echo) {
			errorhandler.New(e, log, lf, obsCfg)
		},
	}
}
