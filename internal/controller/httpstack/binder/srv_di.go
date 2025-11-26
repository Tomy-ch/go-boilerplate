package binder

import (
	"boilerplate-go/internal/controller/httpstack"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
)

// Module は、Binder モジュールを提供します。
func Module() fx.Option {
	return fx.Module("server.binder",
		fx.Provide(
			provideServeConfig,
		),
	)
}

// provideServeConfig は、Binder のサーバー設定を提供します。
func provideServeConfig() httpstack.ServeCfgOut {
	return httpstack.ServeCfgOut{
		SrvCfg: func(e *echo.Echo) {
			New(e)
		},
	}
}
