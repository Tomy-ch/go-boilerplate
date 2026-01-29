// Package inbound は、サーバーのリクエストや入力時の拡張機能に関するDIモジュールを提供します。
package inbound

import (
	"boilerplate-go/internal/controller/httpstack/oapi"
	"boilerplate-go/internal/di/server/extension"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"go.uber.org/fx"
)

const validatorUsePriority = 6

// OpenAPIModule は、OpenAPIバリデーションのミドルウェアを提供するfxモジュールを返します。
func OpenAPIModule() fx.Option {
	return fx.Module("mw.openapi",
		fx.Provide(
			OpenAPIMiddleware,
		),
	)
}

// OpenAPIMiddleware は、バリデーションミドルウェアを提供します。
func OpenAPIMiddleware(
	spec *openapi3.T,
	authFunc openapi3filter.AuthenticationFunc,
) extension.UseMiddlewareOut {
	return extension.UseMiddlewareOut{
		Middleware: extension.UseMiddleware{
			Name:       "openapi",
			Priority:   validatorUsePriority,
			Middleware: oapi.Middleware(spec, authFunc),
		},
	}
}
