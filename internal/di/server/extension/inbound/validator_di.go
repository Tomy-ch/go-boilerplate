// Package inbound は、サーバーのリクエストや入力時の拡張機能に関するDIモジュールを提供します。
package inbound

import (
	"boilerplate-go/internal/controller/httpstack/validator"
	"boilerplate-go/internal/di/server/extension"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/fx"
)

const validatorUsePriority = 6

// ValidatorModule は、OpenAPIバリデーションのミドルウェアを提供するfxモジュールを返します。
func ValidatorModule() fx.Option {
	return fx.Module("mw.validator",
		fx.Provide(
			ValidatorMiddleware,
		),
	)
}

// ValidatorMiddleware は、バリデーションミドルウェアを提供します。
func ValidatorMiddleware(spec *openapi3.T) extension.UseMiddlewareOut {
	return extension.UseMiddlewareOut{
		Middleware: extension.UseMiddleware{
			Name:       "validator",
			Priority:   validatorUsePriority,
			Middleware: validator.Middleware(spec),
		},
	}
}
