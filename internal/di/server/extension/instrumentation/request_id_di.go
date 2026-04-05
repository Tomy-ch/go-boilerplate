package instrumentation

import (
	"go-boilerplate/internal/controller/httpstack/requestid"
	"go-boilerplate/internal/di/server/extension"

	"go.uber.org/fx"
)

const requestIDPriority = 1

// RequestIDModule は、リクエストID制御のミドルウェアを提供するfxモジュールを返します。
func RequestIDModule() fx.Option {
	return fx.Module("mw.requestid",
		fx.Provide(
			RequestIDMiddleware,
		),
	)
}

// RequestIDMiddleware は、リクエストIDの生成ミドルウェアを提供します。
func RequestIDMiddleware() extension.UseMiddlewareOut {
	return extension.UseMiddlewareOut{
		Middleware: extension.UseMiddleware{
			Name:       "requestid",
			Priority:   requestIDPriority,
			Middleware: requestid.Middleware(),
		},
	}
}
