package conv

import (
	"testing"

	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/require"
)

func TestUUID(t *testing.T) {
	t.Parallel()

	src, err := uuid.New()
	require.NoError(t, err)

	got := UUID(src.ToPrimitive())
	require.Equal(t, src.String(), got.String())
}

func Test_mustParseUUID(t *testing.T) {
	t.Parallel()

	t.Run("正常系_有効なUUID文字列を変換できる", func(t *testing.T) {
		t.Parallel()
		src, err := uuid.New()
		require.NoError(t, err)

		got := mustParseUUID(src.String())
		require.Equal(t, src.String(), got.String())
	})

	t.Run("異常系_不正なUUID文字列の場合はpanicする", func(t *testing.T) {
		t.Parallel()
		require.Panics(t, func() {
			mustParseUUID("not-a-uuid")
		})
	})
}
