package errorhandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/controller/httpstack/oapi/validator"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_buildDetailExposureMap_matchesContract は、opt-in マップ(policy が details プロパティの有無で
// 導出)が、OpenAPI 契約(error レスポンスが ErrorResponseWithDetails コンポーネントを参照するか)と
// 1:1 で一致することを固定する契約テストです。
//
// 期待集合は policy とは独立した方法で導出します: 各 operation の error レスポンス JSON スキーマが
// components.schemas.ErrorResponseWithDetails と同一(kin-openapi は $ref を共有ポインタへ解決する)
// かをポインタ同一性で判定します。両者が一致することで「details プロパティ有無 ⇔ WithDetails 参照」の
// 暗黙依存(ADR-0044)が壊れていないことを保証します。
func Test_buildDetailExposureMap_matchesContract(t *testing.T) {
	t.Parallel()

	spec, err := validator.GetValidator()
	require.NoError(t, err)
	require.NotNil(t, spec.Components)

	withDetails := spec.Components.Schemas["ErrorResponseWithDetails"]
	require.NotNil(t, withDetails, "ErrorResponseWithDetails コンポーネントが spec に存在すること")
	require.NotNil(t, withDetails.Value)

	// 契約から独立導出(ポインタ同一性)した期待集合と、policy の property 判定を突き合わせる。
	expected := operationsReferencingSchema(spec, withDetails.Value)
	actual := buildDetailExposureMap(spec)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("opt-inマップが契約(WithDetails参照)と1:1で一致する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, expected, actual)
		})

		t.Run("スキーマ分割が実際に効いている(空でない)", func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, expected)
		})

		t.Run("details を返す既知の operation が opt-in に含まれる", func(t *testing.T) {
			t.Parallel()
			assert.True(t, actual["PostUsers"])
			assert.True(t, actual["PutUsersDetail"])
			assert.True(t, actual["PatchUsersDetail"])
		})
	})
}

// operationsReferencingSchema は、いずれかの error レスポンス(4xx/5xx)の JSON スキーマが
// target と同一(kin-openapi は $ref を共有ポインタへ解決)である operation の集合を返します。
// policy の details プロパティ判定とは独立した契約導出です。
func operationsReferencingSchema(spec *openapi3.T, target *openapi3.Schema) map[string]bool {
	found := make(map[string]bool)
	for _, item := range spec.Paths.Map() {
		for _, op := range item.Operations() {
			if op.OperationID != "" && op.Responses != nil && operationReferencesSchema(op, target) {
				found[op.OperationID] = true
			}
		}
	}
	return found
}

// operationReferencesSchema は、operation の error レスポンスのいずれかが target スキーマを参照するかを返します。
func operationReferencesSchema(op *openapi3.Operation, target *openapi3.Schema) bool {
	for status, respRef := range op.Responses.Map() {
		if isErrorStatusCode(status) && responseReferencesSchema(respRef, target) {
			return true
		}
	}
	return false
}

// responseReferencesSchema は、レスポンスの application/json スキーマが target と同一かを返します。
func responseReferencesSchema(respRef *openapi3.ResponseRef, target *openapi3.Schema) bool {
	if respRef == nil || respRef.Value == nil {
		return false
	}
	mediaType := respRef.Value.Content.Get("application/json")
	if mediaType == nil || mediaType.Schema == nil {
		return false
	}
	return mediaType.Schema.Value == target
}

func Test_openAPIDetailPolicy_Allows(t *testing.T) {
	t.Parallel()

	spec, err := validator.GetValidator()
	require.NoError(t, err)
	built, err := NewOpenAPIDetailPolicy(spec)
	require.NoError(t, err)
	policy, ok := built.(*openAPIDetailPolicy)
	require.True(t, ok)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("opt-in済みoperationは許可される", func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/users", nil)
			assert.True(t, policy.Allows(req))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未opt-inのoperationはfail-closedで拒否される", func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/users/123e4567-e89b-12d3-a456-426614174000", nil)
			assert.False(t, policy.Allows(req))
		})

		t.Run("ルート解決に失敗するパスはfail-closedで拒否される", func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/no/such/path", nil)
			assert.False(t, policy.Allows(req))
		})
	})
}

func Test_operationExposesDetails(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("errorレスポンスにdetailsを持つoperationはtrue", func(t *testing.T) {
			t.Parallel()
			withDetails := openapi3.NewResponse().WithJSONSchema(
				openapi3.NewObjectSchema().WithProperty(detailsPropertyName, openapi3.NewArraySchema()),
			)
			responses := openapi3.NewResponses()
			responses.Set("422", &openapi3.ResponseRef{Value: withDetails})

			assert.True(t, operationExposesDetails(&openapi3.Operation{Responses: responses}))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("エラーステータスでないレスポンスのdetailsは無視される", func(t *testing.T) {
			t.Parallel()
			withDetails := openapi3.NewResponse().WithJSONSchema(
				openapi3.NewObjectSchema().WithProperty(detailsPropertyName, openapi3.NewArraySchema()),
			)
			responses := openapi3.NewResponses()
			responses.Set("200", &openapi3.ResponseRef{Value: withDetails})

			assert.False(t, operationExposesDetails(&openapi3.Operation{Responses: responses}))
		})

		t.Run("errorレスポンスにdetailsが無ければfalse", func(t *testing.T) {
			t.Parallel()
			base := openapi3.NewResponse().WithJSONSchema(openapi3.NewObjectSchema())
			responses := openapi3.NewResponses()
			responses.Set("404", &openapi3.ResponseRef{Value: base})

			assert.False(t, operationExposesDetails(&openapi3.Operation{Responses: responses}))
		})
	})
}

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

		t.Run("スキーマ値が nil の場合 false", func(t *testing.T) {
			t.Parallel()
			resp := openapi3.NewResponse()
			resp.Content = openapi3.Content{"application/json": &openapi3.MediaType{Schema: &openapi3.SchemaRef{}}}
			assert.False(t, responseHasDetailsProperty(&openapi3.ResponseRef{Value: resp}))
		})
	})
}

func Test_schemaHasDetailsProperty(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("トップレベルに details プロパティがある場合 true", func(t *testing.T) {
			t.Parallel()
			schema := openapi3.NewObjectSchema().WithProperty(detailsPropertyName, openapi3.NewArraySchema())
			assert.True(t, schemaHasDetailsProperty(schema))
		})

		t.Run("allOf 要素に details プロパティがある場合 true", func(t *testing.T) {
			t.Parallel()
			schema := openapi3.NewObjectSchema()
			schema.AllOf = openapi3.SchemaRefs{
				openapi3.NewSchemaRef("", openapi3.NewObjectSchema()),
				openapi3.NewSchemaRef("", openapi3.NewObjectSchema().WithProperty(detailsPropertyName, openapi3.NewArraySchema())),
			}
			assert.True(t, schemaHasDetailsProperty(schema))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nil の場合 false", func(t *testing.T) {
			t.Parallel()
			assert.False(t, schemaHasDetailsProperty(nil))
		})

		t.Run("details プロパティが無い場合 false", func(t *testing.T) {
			t.Parallel()
			assert.False(t, schemaHasDetailsProperty(openapi3.NewObjectSchema()))
		})

		t.Run("allOf に nil 要素が含まれても details が無ければ false", func(t *testing.T) {
			t.Parallel()
			schema := openapi3.NewObjectSchema()
			schema.AllOf = openapi3.SchemaRefs{nil, openapi3.NewSchemaRef("", openapi3.NewObjectSchema())}
			assert.False(t, schemaHasDetailsProperty(schema))
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

func Test_newHostAgnosticRouter(t *testing.T) {
	t.Parallel()

	newServersSpec := func(t *testing.T) *openapi3.T {
		t.Helper()

		paths := openapi3.NewPaths()
		paths.Set("/items", &openapi3.PathItem{Get: &openapi3.Operation{OperationID: "GetItems"}})
		return &openapi3.T{
			OpenAPI: "3.0.0",
			Info:    &openapi3.Info{Title: "host-agnostic", Version: "1"},
			Paths:   paths,
			Servers: openapi3.Servers{{URL: "http://spec-host.example"}},
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("specのserversと異なるHostのリクエストでもルートを解決できる", func(t *testing.T) {
			t.Parallel()

			router, err := newHostAgnosticRouter(newServersSpec(t))
			require.NoError(t, err)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://request-host.example/items", nil)
			route, _, err := router.FindRoute(req)
			require.NoError(t, err)
			assert.Equal(t, "/items", route.Path)
		})

		t.Run("引数のspecのserversは書き換えられない", func(t *testing.T) {
			t.Parallel()

			spec := newServersSpec(t)
			_, err := newHostAgnosticRouter(spec)
			require.NoError(t, err)

			require.Len(t, spec.Servers, 1)
			assert.Equal(t, "http://spec-host.example", spec.Servers[0].URL)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("router構築に失敗するspecの場合、エラーを握り潰さず返す", func(t *testing.T) {
			t.Parallel()

			spec := &openapi3.T{Paths: openapi3.NewPaths()}
			spec.Paths.Set("/{id:(}", &openapi3.PathItem{Get: openapi3.NewOperation()})

			router, err := newHostAgnosticRouter(spec)
			require.Error(t, err)
			assert.Nil(t, router)
		})
	})
}
