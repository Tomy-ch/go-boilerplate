// Package oapi は、OpenAPIに関連するHTTPスタックの機能を提供します。
package oapi

import (
	"go-boilerplate/internal/controller/ctxhelper"

	echomw "github.com/labstack/echo/v4/middleware"
	oapimw "github.com/oapi-codegen/echo-middleware"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v4"
)

// Middleware は、OpenAPI スキーマに基づくリクエストバリデーション（認証は authFunc 経由）を行うミドルウェアを返します。
func Middleware(
	spec *openapi3.T,
	skipper echomw.Skipper,
	authFunc openapi3filter.AuthenticationFunc,
) echo.MiddlewareFunc {
	oapiValidator := oapimw.OapiRequestValidatorWithOptions(spec, &oapimw.Options{
		SilenceServersWarning: true,
		Skipper:               skipper,
		Options: openapi3filter.Options{
			AuthenticationFunc: authFunc,
		},
	})

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			req = req.WithContext(ctxhelper.WithAuthn(req.Context()))
			c.SetRequest(req)

			return oapiValidator(next)(c)
		}
	}
}
