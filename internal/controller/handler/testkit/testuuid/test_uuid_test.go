package testuuid

import (
	"testing"

	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
)

func TestRequestUUID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ゼロ値でないUUIDが生成される", func(t *testing.T) {
			t.Parallel()
			got := RequestUUID(t)
			assert.NotEqual(t, types.UUID{}, got)
		})

		t.Run("呼び出しごとに異なるUUIDが生成される", func(t *testing.T) {
			t.Parallel()
			first := RequestUUID(t)
			second := RequestUUID(t)
			assert.NotEqual(t, first, second)
		})
	})
}
