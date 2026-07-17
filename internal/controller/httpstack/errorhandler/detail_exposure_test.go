package errorhandler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/controller/httpstack/errorhandler"
	"go-boilerplate/internal/controller/httpstack/oapi/validator"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOpenAPIDetailPolicy(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な spec からポリシーが構築される", func(t *testing.T) {
			t.Parallel()
			spec, err := validator.GetValidator()
			require.NoError(t, err)

			policy, err := errorhandler.NewOpenAPIDetailPolicy(spec)
			require.NoError(t, err)
			assert.NotNil(t, policy)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("router 構築に失敗する spec ではエラーを返す", func(t *testing.T) {
			t.Parallel()
			// gorilla/mux がコンパイルできない正規表現をパス変数に含む spec を渡すと
			// NewRouter が失敗し、その error がそのまま伝播することを検証する。
			spec := &openapi3.T{Paths: openapi3.NewPaths()}
			spec.Paths.Set("/{id:(}", &openapi3.PathItem{Get: openapi3.NewOperation()})

			policy, err := errorhandler.NewOpenAPIDetailPolicy(spec)
			require.Error(t, err)
			assert.Nil(t, policy)
		})
	})
}

func TestOpenAPIDetailPolicy_Allows(t *testing.T) {
	t.Parallel()

	spec, err := validator.GetValidator()
	require.NoError(t, err)
	policy, err := errorhandler.NewOpenAPIDetailPolicy(spec)
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("details をエラーレスポンスに宣言した operation は許可される", func(t *testing.T) {
			t.Parallel()

			t.Run("POST /v1/users は許可される", func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/users", nil)
				assert.True(t, policy.Allows(req))
			})

			t.Run("PUT /v1/users/{userId} は許可される", func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/v1/users/123e4567-e89b-12d3-a456-426614174000", nil)
				assert.True(t, policy.Allows(req))
			})

			t.Run("PATCH /v1/users/{userId} は許可される", func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/v1/users/123e4567-e89b-12d3-a456-426614174000", nil)
				assert.True(t, policy.Allows(req))
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("details を宣言していない operation は fail-closed で拒否される", func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/users/123e4567-e89b-12d3-a456-426614174000", nil)
			assert.False(t, policy.Allows(req))
		})

		t.Run("ルートに一致しないパスは fail-closed で拒否される", func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/no/such/path", nil)
			assert.False(t, policy.Allows(req))
		})
	})
}
