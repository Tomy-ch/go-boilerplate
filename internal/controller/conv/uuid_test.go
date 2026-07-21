package conv

import (
	"testing"

	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUUID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("OpenAPI生成UUIDをドメインUUIDへ変換できる", func(t *testing.T) {
			t.Parallel()
			src, err := uuid.New()
			require.NoError(t, err)

			got := UUID(src.ToPrimitive())
			assert.Equal(t, src.String(), got.String())
		})
	})
}

func TestUUIDPtr(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilの場合はnilを返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, UUIDPtr(nil))
		})

		t.Run("非nilの場合はドメインUUIDへ変換する", func(t *testing.T) {
			t.Parallel()
			src, err := uuid.New()
			require.NoError(t, err)

			p := src.ToPrimitive()
			got := UUIDPtr(&p)
			require.NotNil(t, got)
			assert.Equal(t, src.String(), got.String())
		})
	})
}
