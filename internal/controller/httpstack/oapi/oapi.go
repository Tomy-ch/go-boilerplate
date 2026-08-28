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
// バリデーション実行前にリクエストコンテキストへ authn スロット（ctxhelper.WithAuthn）と stream grant スロット
// （ctxhelper.WithStreamGrant）を注入するため、authFunc はそのスロットへ認証結果を書き込めます。
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
			req = req.WithContext(ctxhelper.WithStreamGrant(ctxhelper.WithAuthn(req.Context())))
			c.SetRequest(req)

			return oapiValidator(failClosed(next))(c)
		}
	}
}

// failClosed は、認証に失敗したリクエストをハンドラへ到達させず、その失敗を返します。
// 認証を任意と宣言した operation でも、提示されたうえで検証に失敗した資格情報は通さない。
// 根拠は ADR-0021 (optional-authentication-fail-closed) と README.md「Fail-closed authentication」。
func failClosed(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if err := ctxhelper.AuthnFailure(c.Request().Context()); err != nil {
			return err
		}
		return next(c)
	}
}
