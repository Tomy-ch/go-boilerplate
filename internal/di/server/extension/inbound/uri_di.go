package inbound

import (
	"boilerplate-go/internal/controller/httpstack/uri"
	"boilerplate-go/internal/di/server/extension"

	"go.uber.org/fx"
)

const uriPrePriority = 1

// URIModule は、URI制御のミドルウェアを提供するfxモジュールを返します。
func URIModule() fx.Option {
	return fx.Module("mw.uri",
		fx.Provide(
			URIPreMiddleware,
		),
	)
}

// URIPreMiddleware は、バリデーションミドルウェアを提供します。
func URIPreMiddleware() extension.PreMiddlewareOut {
	return extension.PreMiddlewareOut{
		Middleware: extension.PreMiddleware{
			Name:       "uri",
			Priority:   uriPrePriority,
			Middleware: uri.Middleware(),
		},
	}
}
