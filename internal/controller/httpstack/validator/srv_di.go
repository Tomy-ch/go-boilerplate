package validator

import (
	"boilerplate-go/internal/controller/httpstack"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
)

// Module は、Validator モジュールを提供します。
func Module() fx.Option {
	return fx.Module("server.validator",
		fx.Provide(
			func() httpstack.ServeCfgOut {
				return httpstack.ServeCfgOut{
					SrvCfg: func(e *echo.Echo) {
						New(e)
					},
				}
			},
		),
	)
}
