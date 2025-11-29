package validator

import (
	"boilerplate-go/internal/controller/httpstack"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/fx"
)

const priority = 1

// Module は、OpenAPIバリデーションのミドルウェアを提供するfxモジュールを返します。
func Module() fx.Option {
	return fx.Module("mw.validator",
		fx.Provide(
			UseMiddleware,
		),
	)
}

// UseMiddleware は、バリデーションミドルウェアを提供します。
func UseMiddleware(spec *openapi3.T) httpstack.UseMiddlewareOut {
	return httpstack.UseMiddlewareOut{
		Middleware: httpstack.UseMiddleware{
			Priority:   priority,
			Middleware: Middleware(spec),
		},
	}
}

// CoreModule は、ルーティング時に自動で解決されるバリデーションのコア機能部分を提供するfxモジュールを返します。
func CoreModule() fx.Option {
	return fx.Module("validator.core",
		fx.Provide(
			GetValidator,
		),
	)
}
