// Package decoration は、サーバー起動時の装飾に関するDIモジュールを提供します。
package decoration

import (
	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/httpstack/banner"
	"boilerplate-go/internal/di/server/extension"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
)

// BannerModule は、バナー表示を制御するためのモジュールです。
func BannerModule() fx.Option {
	return fx.Module("server.banner",
		fx.Provide(
			provideBannerServeConfig,
		),
	)
}

// provideBannerServeConfig は、バナー表示のサーバー設定を提供します。
func provideBannerServeConfig(appCfg *config.ApplicationConfig) extension.ServeCfgOut {
	return extension.ServeCfgOut{
		SrvCfg: func(e *echo.Echo) {
			banner.New(e, appCfg)
		},
	}
}
