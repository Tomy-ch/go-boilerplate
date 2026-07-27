package conv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInt32(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("intをint32へ変換する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, int32(42), Int32(42))
		})

		t.Run("ゼロ値を変換する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, int32(0), Int32(0))
		})
	})
}
