package conv

import (
	"testing"

	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUUID(t *testing.T) {
	t.Parallel()

	t.Run("正常系_OpenAPI生成UUIDをドメインUUIDへ変換できる", func(t *testing.T) {
		t.Parallel()
		src, err := uuid.New()
		require.NoError(t, err)

		got := UUID(src.ToPrimitive())
		assert.Equal(t, src.String(), got.String())
	})
}
