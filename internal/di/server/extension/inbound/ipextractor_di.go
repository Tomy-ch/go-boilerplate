package inbound

import (
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/ipextractor"
	"go-boilerplate/internal/di/server/extension"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
)

// IPExtractorModule は、IP Extractor モジュールを提供します。
func IPExtractorModule() fx.Option {
	return fx.Module("server.ipextractor",
		fx.Provide(
			provideIPExtractorServeConfig,
		),
	)
}

// provideIPExtractorServeConfig は、IP Extractor のサーバー設定を提供します。
func provideIPExtractorServeConfig(
	appCfg *config.ApplicationConfig, secCfg *config.SecurityConfig,
) extension.ServeCfgOut {
	return extension.ServeCfgOut{
		SrvCfg: func(e *echo.Echo) {
			ipextractor.New(e, appCfg, secCfg)
		},
	}
}
