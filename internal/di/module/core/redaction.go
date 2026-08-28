package core

import (
	"go-boilerplate/internal/controller/httpstack/redaction"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/fx"
)

// RedactionModule は、ログへ出す前に資格情報を取り除く Redactor を提供する fx モジュールを返します。
// 秘匿する名前は OpenAPI spec の securityScheme から導出するため、ValidatorModule と同じ spec を受け取ります。
func RedactionModule() fx.Option {
	return fx.Module("core.redaction",
		fx.Provide(
			provideRedactor,
		),
	)
}

// provideRedactor は、spec の query apiKey scheme から Redactor を構築して fx に提供します。
func provideRedactor(spec *openapi3.T) redaction.Redactor {
	return redaction.FromSpec(spec)
}
