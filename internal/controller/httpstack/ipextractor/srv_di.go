package ipextractor

import (
	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/httpstack"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
)

// Module は、IP Extractor モジュールを提供します。
func Module() fx.Option {
	return fx.Module("server.ipextractor",
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
