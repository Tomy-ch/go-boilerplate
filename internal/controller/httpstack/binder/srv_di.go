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
			func() httpstack.ServeCfgOut {
				return httpstack.ServeCfgOut{
					SrvCfg: func(e *echo.Echo) { New(e) },
				}
			},
		),
	)
}
