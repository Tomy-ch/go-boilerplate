package banner

import (
	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/httpstack"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
)

// Module は、バナー表示を制御するためのモジュールです。
func Module() fx.Option {
	return fx.Module("server.banner",
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
