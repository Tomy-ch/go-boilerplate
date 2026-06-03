package testuuid

import (
	"testing"

	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"
)

func TestRequestUUID(t *testing.T) {
	t.Parallel()

	t.Run("正常系_ゼロ値でないUUIDが生成される", func(t *testing.T) {
		t.Parallel()
		got := RequestUUID(t)
		require.NotEqual(t, types.UUID{}, got)
	})

	t.Run("正常系_呼び出しごとに異なるUUIDが生成される", func(t *testing.T) {
		t.Parallel()
		first := RequestUUID(t)
		second := RequestUUID(t)
		require.NotEqual(t, first, second)
	})
}
