package errorhandler

import (
	"boilerplate-go/internal/controller/httpstack"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func Module() fx.Option {
	return fx.Module("server.errorhandler",
		fx.Provide(
			func(z *zap.Logger) httpstack.ServeCfgOut {
				return httpstack.ServeCfgOut{
					SrvCfg: func(e *echo.Echo) {
						New(e, z)
					},
				}
			},
		),
	)
}
