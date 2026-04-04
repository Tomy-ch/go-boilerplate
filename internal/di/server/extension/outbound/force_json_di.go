package outbound

import (
	"go-boilerplate/internal/controller/httpstack/forcejson"
	"go-boilerplate/internal/di/server/extension"

	"go.uber.org/fx"
)

const forceJSONPriority = 7

// ForceJSONModule は、JOINの強制制御のミドルウェアを提供するfxモジュールを返します。
func ForceJSONModule() fx.Option {
	return fx.Module("mw.forcejson",
		fx.Provide(
			ForceJSONMiddleware,
		),
	)
}

// ForceJSONMiddleware は、JOINの強制制御ミドルウェアを提供します。
func ForceJSONMiddleware() extension.UseMiddlewareOut {
	return extension.UseMiddlewareOut{
		Middleware: extension.UseMiddleware{
			Name:       "forcejson",
			Priority:   forceJSONPriority,
			Middleware: forcejson.Middleware(),
		},
	}
}
