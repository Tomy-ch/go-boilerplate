package defaultport

import (
	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/httpstack"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
)

// Module は、ポート設定を制御するためのモジュールです。
func Module() fx.Option {
	return fx.Module("server.port",
		fx.Provide(
			provideServeConfig,
		),
	)
}

// provideServeConfig は、ポート設定を提供します。
func provideServeConfig(cfg *config.Config) httpstack.ServeCfgOut {
	return httpstack.ServeCfgOut{
		SrvCfg: func(e *echo.Echo) {
			New(e, cfg)
		},
	}
}
