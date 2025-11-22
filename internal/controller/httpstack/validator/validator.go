//go:generate oapi-codegen --package=gen --generate=spec -o ./gen/validate.gen.go /app/openapi/openapi.gen.yaml

// Package validator は、リクエストの検証を行うミドルウェアを提供します。
package validator

import (
	"boilerplate-go/internal/controller/httpstack/validator/gen"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	middleware "github.com/oapi-codegen/echo-middleware"
)

// Middleware は、リクエストの検証を行うミドルウェアを提供します。
func Middleware(validator *openapi3.T) echo.MiddlewareFunc {
	return middleware.OapiRequestValidator(validator)
}

func GetValidator() (*openapi3.T, error) {
	return gen.GetSwagger()
}
