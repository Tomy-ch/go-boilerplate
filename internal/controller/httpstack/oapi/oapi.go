// Package oapi は、OpenAPIに関連するHTTPスタックの機能を提供します。
package oapi

import (
	"go-boilerplate/internal/controller/ctxhelper"

	echomw "github.com/labstack/echo/v5/middleware"
	oapimw "github.com/oapi-codegen/echo-v5-middleware"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v5"
)

// Middleware は、OpenAPI スキーマに基づくリクエストバリデーション（認証は authFunc 経由）を行うミドルウェアを返します。
// バリデーション実行前にリクエストコンテキストへ authn スロット（ctxhelper.WithAuthn）を注入するため、authFunc はそのスロットへ認証結果を書き込めます。
// 認証が失敗したリクエストは、spec がそれを任意と宣言していてもハンドラへ到達しません（failClosed）。
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
		return func(c *echo.Context) error {
			req := c.Request()
			req = req.WithContext(ctxhelper.WithAuthn(req.Context()))
			c.SetRequest(req)

			return oapiValidator(failClosed(next))(c)
		}
	}
}

// failClosed は、認証に失敗したリクエストをハンドラへ到達させず、その失敗を返します。
//
// spec が複数の security requirement を並べた operation では、そのうち 1 つでも満たされれば
// バリデーションは成功する。資格情報を要求しない requirement は常に満たされるため、
// 提示された資格情報の検証失敗はバリデーションの結果に現れず、認証されていない主体が
// ハンドラへ到達する。ここで失敗を拾い直すことで、認証を任意とする宣言が
// 「検証に失敗しても通す」という意味にならないようにする。
func failClosed(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if err := ctxhelper.AuthnFailure(c.Request().Context()); err != nil {
			return err
		}
		return next(c)
	}
}
