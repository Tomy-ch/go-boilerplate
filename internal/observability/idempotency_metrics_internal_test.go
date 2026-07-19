package observability

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_normalizeOperationID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非空の operationID はそのまま返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "CreateUser", normalizeOperationID("CreateUser"))
		})

		t.Run("空の operationID は unknown へ丸める", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "unknown", normalizeOperationID(""))
		})
	})
}
