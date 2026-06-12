package conv

import (
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Email は、OpenAPI 生成の Email を文字列へ変換します。
func Email(e openapi_types.Email) string {
	return string(e)
}

// EmailPtr は、任意の OpenAPI 生成 Email を文字列ポインタへ変換します（nil は nil を返す）。
func EmailPtr(e *openapi_types.Email) *string {
	if e == nil {
		return nil
	}
	s := string(*e)
	return &s
}
