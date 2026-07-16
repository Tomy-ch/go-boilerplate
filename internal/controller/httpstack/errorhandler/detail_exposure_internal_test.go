package errorhandler

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
)

func Test_isErrorStatusCode(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("4xx はエラーステータス", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isErrorStatusCode("422"))
		})

		t.Run("5xx はエラーステータス", func(t *testing.T) {
			t.Parallel()
			assert.True(t, isErrorStatusCode("500"))
		})

		t.Run("2xx はエラーステータスではない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isErrorStatusCode("200"))
		})

		t.Run("数値でないワイルドカードはエラーステータスではない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, isErrorStatusCode("default"))
		})
	})
}

func Test_responseHasDetailsProperty(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JSONスキーマに details プロパティがある場合 true", func(t *testing.T) {
			t.Parallel()
			resp := openapi3.NewResponse().WithJSONSchema(
				openapi3.NewObjectSchema().WithProperty(detailsPropertyName, openapi3.NewArraySchema()),
			)
			assert.True(t, responseHasDetailsProperty(&openapi3.ResponseRef{Value: resp}))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("respRef が nil の場合 false", func(t *testing.T) {
			t.Parallel()
			assert.False(t, responseHasDetailsProperty(nil))
		})

		t.Run("Value が nil の場合 false", func(t *testing.T) {
			t.Parallel()
			assert.False(t, responseHasDetailsProperty(&openapi3.ResponseRef{}))
		})

		t.Run("application/json コンテンツが無い場合 false", func(t *testing.T) {
			t.Parallel()
			assert.False(t, responseHasDetailsProperty(&openapi3.ResponseRef{Value: openapi3.NewResponse()}))
		})

		t.Run("JSONスキーマに details プロパティが無い場合 false", func(t *testing.T) {
			t.Parallel()
			resp := openapi3.NewResponse().WithJSONSchema(openapi3.NewObjectSchema())
			assert.False(t, responseHasDetailsProperty(&openapi3.ResponseRef{Value: resp}))
		})
	})
}

func Test_buildDetailExposureMap(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("エラーレスポンスに details を持つ operation だけが true になる", func(t *testing.T) {
			t.Parallel()
			withDetails := openapi3.NewResponse().WithJSONSchema(
				openapi3.NewObjectSchema().WithProperty(detailsPropertyName, openapi3.NewArraySchema()),
			)
			base := openapi3.NewResponse().WithJSONSchema(openapi3.NewObjectSchema())

			exposed := openapi3.NewResponses()
			exposed.Set("422", &openapi3.ResponseRef{Value: withDetails})
			notExposed := openapi3.NewResponses()
			notExposed.Set("404", &openapi3.ResponseRef{Value: base})

			spec := &openapi3.T{Paths: openapi3.NewPaths()}
			spec.Paths.Set("/exposed", &openapi3.PathItem{Get: &openapi3.Operation{OperationID: "Exposed", Responses: exposed}})
			spec.Paths.Set("/plain", &openapi3.PathItem{Get: &openapi3.Operation{OperationID: "Plain", Responses: notExposed}})

			got := buildDetailExposureMap(spec)
			assert.True(t, got["Exposed"])
			assert.False(t, got["Plain"])
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Paths が nil の場合は空マップ", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, buildDetailExposureMap(&openapi3.T{}))
		})

		t.Run("operationId が空の operation は無視される", func(t *testing.T) {
			t.Parallel()
			withDetails := openapi3.NewResponse().WithJSONSchema(
				openapi3.NewObjectSchema().WithProperty(detailsPropertyName, openapi3.NewArraySchema()),
			)
			responses := openapi3.NewResponses()
			responses.Set("422", &openapi3.ResponseRef{Value: withDetails})

			spec := &openapi3.T{Paths: openapi3.NewPaths()}
			spec.Paths.Set("/anon", &openapi3.PathItem{Get: &openapi3.Operation{OperationID: "", Responses: responses}})

			assert.Empty(t, buildDetailExposureMap(spec))
		})
	})
}
