//go:generate oapi-codegen --package=gen --generate=spec -o ./gen/validate.gen.go /app/openapi/openapi.gen.yaml

// Package validator は、OpenAPI 仕様(spec)を読み込み提供します（検証 mw 生成は oapi が担当）。
package validator

import (
	"go-boilerplate/internal/controller/httpstack/oapi/validator/gen"

	"github.com/getkin/kin-openapi/openapi3"
)

// GetValidator は、OpenAPI仕様(spec)を返します。
func GetValidator() (*openapi3.T, error) {
	return gen.GetSwagger()
}
