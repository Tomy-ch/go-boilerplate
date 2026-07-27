package conv

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeOrZero(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非nilの場合は値を返す", func(t *testing.T) {
			t.Parallel()
			tm := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
			assert.Equal(t, tm, TimeOrZero(&tm))
		})

		t.Run("nilの場合はゼロ値を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, time.Time{}, TimeOrZero(nil))
		})
	})
}
