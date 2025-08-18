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
			func(cfg *config.Config) httpstack.ServeCfgOut {
				return httpstack.ServeCfgOut{
					SrvCfg: func(e *echo.Echo) {
						New(e, cfg)
					},
				}
			},
		),
	)
}
