// Package nonprod は、非本番環境向けのサーバー拡張機能に関するDIモジュールを提供します。
package nonprod

import (
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/debugmode"
	"go-boilerplate/internal/di/server/extension"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
)

// DebugModeModule は、デバッグモードを制御するためのモジュールです。
func DebugModeModule() fx.Option {
	return fx.Module("server.debugmode",
		fx.Provide(
			provideDebugModeServeConfig,
		),
	)
}

// provideDebugModeServeConfig は、デバッグモードのサーバー設定を提供します。
func provideDebugModeServeConfig(appCfg *config.ApplicationConfig) extension.ServeCfgOut {
	return extension.ServeCfgOut{
		SrvCfg: func(e *echo.Echo) {
			debugmode.New(e, appCfg)
		},
	}
}
