package errorhandler

import (
	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/httpstack"
	"boilerplate-go/internal/logging"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
)

// Module は、Error Handler モジュールを提供します。
func Module() fx.Option {
	return fx.Module("server.errorhandler",
		fx.Provide(
			provideServeConfig,
		),
	)
}

// provideServeConfig は、Error Handler のサーバー設定を提供します。
func provideServeConfig(log logging.Logger, lf logging.LogFieldBuilder, obsCfg *config.ObservabilityConfig) httpstack.ServeCfgOut {
	return httpstack.ServeCfgOut{
		SrvCfg: func(e *echo.Echo) {
			New(e, log, lf, obsCfg)
		},
	}
}
