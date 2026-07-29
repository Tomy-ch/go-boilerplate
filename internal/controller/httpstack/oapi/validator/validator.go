//go:generate oapi-codegen --package=gen --generate=spec -o ./gen/validate.gen.go /app/openapi/openapi.gen.yaml

// Package validator は、OpenAPI 仕様(spec)を読み込み提供します（検証 mw 生成は oapi が担当）。
package validator

import (
	"go-boilerplate/internal/controller/httpstack/oapi/validator/gen"

	"github.com/getkin/kin-openapi/openapi3"
)

// GetValidator は、OpenAPI仕様(spec)を返します。
// servers は公開 URL の記述であって待ち受け先ではないため、経路解決を Host 非依存にするべく除去します
// (servers を残すと gorillamux が Host マッチを行い、proxy 配下や 8080 以外のポートで待ち受けた場合に
// 全リクエストが「該当 operation なし」で 404 に倒れる)。
func GetValidator() (*openapi3.T, error) {
	spec, err := gen.GetSpec()
	if err != nil {
		return nil, err
	}
	spec.Servers = nil

	return spec, nil
}
