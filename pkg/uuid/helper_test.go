package uuid

import (
	"testing"

	guuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_fromGoogle(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("google.UUIDから同値のUUIDへ変換する", func(t *testing.T) {
			t.Parallel()
			g := guuid.New()
			u := fromGoogle(g)
			assert.Equal(t, g.String(), u.String())
		})
	})
}

func Test_toGoogle(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("UUIDから同値のgoogle.UUIDへ変換する", func(t *testing.T) {
			t.Parallel()
			uuid, err := New()
			require.NoError(t, err)

			g := toGoogle(uuid)
			assert.Equal(t, uuid.String(), g.String())
		})
	})
}
