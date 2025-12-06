package decoration

import (
	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/httpstack/defaultport"
	"boilerplate-go/internal/di/server/extension"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
)

// DefaultPortModule は、ポート設定を制御するためのモジュールです。
func DefaultPortModule() fx.Option {
	return fx.Module("server.port",
		fx.Provide(
			provideDefaultPortServeConfig,
		),
	)
}

// provideDefaultPortServeConfig は、ポート設定を提供します。
func provideDefaultPortServeConfig(appCfg *config.ApplicationConfig) extension.ServeCfgOut {
	return extension.ServeCfgOut{
		SrvCfg: func(e *echo.Echo) {
			defaultport.New(e, appCfg)
		},
	}
}
