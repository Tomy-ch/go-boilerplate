package conv

import (
	"testing"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmail(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("OpenAPIのEmail型を文字列へ変換する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "a@example.com", Email(openapi_types.Email("a@example.com")))
		})
	})
}

func TestEmailPtr(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilの場合はnilを返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, EmailPtr(nil))
		})

		t.Run("値がある場合は文字列ポインタを返す", func(t *testing.T) {
			t.Parallel()
			e := openapi_types.Email("b@example.com")
			got := EmailPtr(&e)
			require.NotNil(t, got)
			assert.Equal(t, "b@example.com", *got)
		})
	})
}
