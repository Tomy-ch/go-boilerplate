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
			provideServeConfig,
		),
	)
}

// provideServeConfig は、IP Extractor のサーバー設定を提供します。
func provideServeConfig(cfg *config.Config) httpstack.ServeCfgOut {
	return httpstack.ServeCfgOut{
		SrvCfg: func(e *echo.Echo) {
			New(e, cfg)
		},
	}
}
