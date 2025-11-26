package debugmode

import (
	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/httpstack"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
)

// Module は、デバッグモードを制御するためのモジュールです。
func Module() fx.Option {
	return fx.Module("server.debugmode",
		fx.Provide(
			provideServeConfig,
		),
	)
}

// provideServeConfig は、デバッグモードのサーバー設定を提供します。
func provideServeConfig(cfg *config.Config) httpstack.ServeCfgOut {
	return httpstack.ServeCfgOut{
		SrvCfg: func(e *echo.Echo) {
			New(e, cfg)
		},
	}
}
