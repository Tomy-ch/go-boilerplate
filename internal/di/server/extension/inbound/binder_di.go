package inbound

import (
	"boilerplate-go/internal/controller/httpstack/binder"
	"boilerplate-go/internal/di/server/extension"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
)

// BinderModule は、Binder モジュールを提供します。
func BinderModule() fx.Option {
	return fx.Module("server.binder",
		fx.Provide(
			provideBinderServeConfig,
		),
	)
}

// provideBinderServeConfig は、Binder のサーバー設定を提供します。
func provideBinderServeConfig() extension.ServeCfgOut {
	return extension.ServeCfgOut{
		SrvCfg: func(e *echo.Echo) {
			binder.New(e)
		},
	}
}
