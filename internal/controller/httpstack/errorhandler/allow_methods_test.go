package errorhandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"go-boilerplate/internal/controller/httpstack/oapi/validator"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pathTemplateParamRe は、パステンプレートの可変部（{userId} 等）にマッチします。
var pathTemplateParamRe = regexp.MustCompile(`\{[^}]+\}`)

func TestNewOpenAPIAllowPolicy(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("実specから構築でき、定義済みパスのAllowを解決できる", func(t *testing.T) {
			t.Parallel()

			spec, err := validator.GetValidator()
			require.NoError(t, err)

			policy, err := NewOpenAPIAllowPolicy(spec)
			require.NoError(t, err)

			req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/v1/prefectures", nil)
			assert.Equal(t, "OPTIONS, GET", policy.Allow(req))
		})

		t.Run("servers付きspecでもHost非依存で解決できる", func(t *testing.T) {
			t.Parallel()

			spec := newAllowSpec(t)
			spec.Servers = openapi3.Servers{{URL: "http://spec-host.example"}}

			policy, err := NewOpenAPIAllowPolicy(spec)
			require.NoError(t, err)

			// spec の servers とも httptest 既定の example.com とも異なる Host を明示し、
			// scheme ではなく Host の不一致で解決できることを確かめる。
			req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "http://request-host.example/items", nil)
			assert.Equal(t, "OPTIONS, GET, POST", policy.Allow(req))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("router構築に失敗するspecの場合、エラーを握り潰さず返す", func(t *testing.T) {
			t.Parallel()

			spec := &openapi3.T{Paths: openapi3.NewPaths()}
			spec.Paths.Set("/{id:(}", &openapi3.PathItem{Get: openapi3.NewOperation()})

			policy, err := NewOpenAPIAllowPolicy(spec)
			require.Error(t, err)
			assert.Nil(t, policy)
		})
	})
}

func Test_openAPIAllowPolicy_Allow(t *testing.T) {
	t.Parallel()

	newPolicy := func(t *testing.T) AllowPolicy {
		t.Helper()
		policy, err := NewOpenAPIAllowPolicy(newAllowSpec(t))
		require.NoError(t, err)
		return policy
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("パステンプレートに定義された全メソッドがOPTIONS付きで返る", func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/items", nil)
			assert.Equal(t, "OPTIONS, GET, POST", newPolicy(t).Allow(req))
		})

		t.Run("可変パスは実値でテンプレートへ解決される", func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/items/42", nil)
			assert.Equal(t, "OPTIONS, DELETE, GET", newPolicy(t).Allow(req))
		})

		t.Run("静的パスは重なる可変パスより優先して解決される", func(t *testing.T) {
			t.Parallel()

			// Echo のルータは /items/{itemId} 側へマッチし得るが、spec 上 /items/me は GET のみ。
			req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/items/me", nil)
			assert.Equal(t, "OPTIONS, GET", newPolicy(t).Allow(req))
		})

		t.Run("先頭のprobeメソッドで解決しないパスも後続のprobeで解決される", func(t *testing.T) {
			t.Parallel()

			// /items/upload は POST のみ。allowProbeMethods 先頭の GET は空振りする。
			req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/items/upload", nil)
			assert.Equal(t, "OPTIONS, POST", newPolicy(t).Allow(req))
		})

		t.Run("probeリスト後方のメソッドだけを定義したパスも解決される", func(t *testing.T) {
			t.Parallel()

			// /items/probe は HEAD のみ。allowProbeMethods の前方5メソッドは全て空振りする。
			req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/items/probe", nil)
			assert.Equal(t, "OPTIONS, HEAD", newPolicy(t).Allow(req))
		})

		t.Run("実specのGETを持たないパスも解決される", func(t *testing.T) {
			t.Parallel()

			spec, err := validator.GetValidator()
			require.NoError(t, err)
			policy, err := NewOpenAPIAllowPolicy(spec)
			require.NoError(t, err)

			req := httptest.NewRequestWithContext(
				context.Background(),
				http.MethodGet,
				"/v1/purchases/123e4567-e89b-12d3-a456-426614174000/cancel",
				nil,
			)
			assert.Equal(t, "OPTIONS, PATCH", policy.Allow(req))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("specのどのルートにも解決しないパスは空文字が返る", func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/no/such/path", nil)
			assert.Empty(t, newPolicy(t).Allow(req))
		})
	})
}

func Test_buildAllowMap(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("パステンプレートごとにOPTIONS始まりのメソッド一覧が組み立てられる", func(t *testing.T) {
			t.Parallel()

			actual := buildAllowMap(newAllowSpec(t))

			assert.Equal(t, "OPTIONS, GET, POST", actual["/items"])
			assert.Equal(t, "OPTIONS, DELETE, GET", actual["/items/{itemId}"])
			assert.Equal(t, "OPTIONS, GET", actual["/items/me"])
		})

		t.Run("operationを1つも持たないパスはOPTIONS単独になる", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "OPTIONS", buildAllowMap(newAllowSpec(t))["/items/meta"])
		})

		t.Run("specにOPTIONSが定義されていても重複しない", func(t *testing.T) {
			t.Parallel()

			spec := newAllowSpec(t)
			spec.Paths.Value("/items").Options = &openapi3.Operation{OperationID: "OptionsItems"}

			assert.Equal(t, "OPTIONS, GET, POST", buildAllowMap(spec)["/items"])
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Pathsを持たないspecは空マップが返る", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, buildAllowMap(&openapi3.T{}))
		})
	})
}

// Test_openAPIAllowPolicy_Allow_coversEverySpecPath は、RFC 9110 §15.5.6 が 405 に MUST とする
// Allow ヘッダーを、実 spec の全パスに対して解決できることを固定する契約テストです。
//
// 405 は「Echo のルータが送出する(ContextKeyHeaderAllow が必ず埋まる)」か「OpenAPI のルータが送出する
// (そのパスが spec に載っていることが前提)」のいずれかなので、spec 上の全パスが解決できることは
// 「405 を返すとき Allow は必ず非空」と等価です。
func Test_openAPIAllowPolicy_Allow_coversEverySpecPath(t *testing.T) {
	t.Parallel()

	spec, err := validator.GetValidator()
	require.NoError(t, err)
	policy, err := NewOpenAPIAllowPolicy(spec)
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("spec上の全パスが未定義メソッドに対してAllowを解決できる", func(t *testing.T) {
			t.Parallel()

			unresolved := make([]string, 0)
			checked := 0
			for path, item := range spec.Paths.Map() {
				method, ok := methodOutsidePathItem(item)
				if !ok {
					continue
				}
				checked++

				url := pathTemplateParamRe.ReplaceAllString(path, "123e4567-e89b-12d3-a456-426614174000")
				req := httptest.NewRequestWithContext(context.Background(), method, url, nil)
				if policy.Allow(req) == "" {
					unresolved = append(unresolved, method+" "+path)
				}
			}

			assert.Empty(t, unresolved, "Allow を解決できないパスは RFC 9110 §15.5.6 の MUST に違反する")
			// spec が空、あるいはパス抽出が壊れて 0 件を検査して通る空振りを防ぐ。
			assert.NotZero(t, checked)
		})
	})
}

// methodOutsidePathItem は、pathItem に定義されていない probe 対象メソッドを1つ返します。
// 全メソッドが定義済みで 405 が起こり得ないパスでは ok=false を返します。
func methodOutsidePathItem(item *openapi3.PathItem) (string, bool) {
	defined := item.Operations()
	if len(defined) == 0 {
		return "", false
	}

	for _, method := range allowProbeMethods {
		if _, ok := defined[method]; !ok {
			return method, true
		}
	}
	return "", false
}

// newAllowSpec は、静的パスと可変パスが重なる構成を持つ検証用の spec を組み立てます。
func newAllowSpec(t *testing.T) *openapi3.T {
	t.Helper()

	paths := openapi3.NewPaths()
	paths.Set("/items", &openapi3.PathItem{
		Get:  &openapi3.Operation{OperationID: "GetItems"},
		Post: &openapi3.Operation{OperationID: "PostItems"},
	})
	paths.Set("/items/me", &openapi3.PathItem{
		Get: &openapi3.Operation{OperationID: "GetItemsMe"},
	})
	paths.Set("/items/{itemId}", &openapi3.PathItem{
		Get:    &openapi3.Operation{OperationID: "GetItemsDetail"},
		Delete: &openapi3.Operation{OperationID: "DeleteItemsDetail"},
	})
	paths.Set("/items/upload", &openapi3.PathItem{
		Post: &openapi3.Operation{OperationID: "PostItemsUpload"},
	})
	paths.Set("/items/meta", &openapi3.PathItem{
		Parameters: openapi3.Parameters{{Value: openapi3.NewQueryParameter("q")}},
	})
	paths.Set("/items/probe", &openapi3.PathItem{
		Head: &openapi3.Operation{OperationID: "HeadItemsProbe"},
	})

	return &openapi3.T{
		OpenAPI: "3.0.0",
		Info:    &openapi3.Info{Title: "allow", Version: "1"},
		Paths:   paths,
	}
}
