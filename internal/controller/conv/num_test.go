package conv

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInt32(t *testing.T) {
	t.Parallel()

	// 分岐の無い恒等キャストのため、代表値ではなく int32 の両境界で恒等性を固定する。
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("int32上限は同値へ恒等変換する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, int32(math.MaxInt32), Int32(math.MaxInt32))
		})

		t.Run("int32下限（負の境界）は同値へ恒等変換する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, int32(math.MinInt32), Int32(math.MinInt32))
		})
	})
}
