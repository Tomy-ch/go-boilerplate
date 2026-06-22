package backoff

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_Exponential_Duration(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("attempt=0 では Initial を返す", func(t *testing.T) {
			t.Parallel()

			e := Exponential{Initial: 100 * time.Millisecond, Max: time.Second, Multiplier: 2}
			assert.Equal(t, 100*time.Millisecond, e.Duration(0))
		})

		t.Run("attempt ごとに Multiplier 倍される", func(t *testing.T) {
			t.Parallel()

			e := Exponential{Initial: 100 * time.Millisecond, Max: time.Second, Multiplier: 2}
			assert.Equal(t, 200*time.Millisecond, e.Duration(1))
			assert.Equal(t, 400*time.Millisecond, e.Duration(2))
		})

		t.Run("Max で頭打ちになる", func(t *testing.T) {
			t.Parallel()

			e := Exponential{Initial: 100 * time.Millisecond, Max: 300 * time.Millisecond, Multiplier: 2}
			assert.Equal(t, 300*time.Millisecond, e.Duration(5))
		})

		t.Run("Multiplier が 1 未満なら 1 として扱う", func(t *testing.T) {
			t.Parallel()

			e := Exponential{Initial: 100 * time.Millisecond, Max: time.Second, Multiplier: 0}
			assert.Equal(t, 100*time.Millisecond, e.Duration(3))
		})

		t.Run("Max 上限なしで attempt が大きくても負値にならず MaxInt64 で頭打ちになる", func(t *testing.T) {
			t.Parallel()

			e := Exponential{Initial: time.Second, Max: 0, Multiplier: 2}
			got := e.Duration(1000)
			assert.Equal(t, time.Duration(math.MaxInt64), got)
			assert.Positive(t, int64(got))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Initial が 0 以下なら 0 を返す", func(t *testing.T) {
			t.Parallel()

			e := Exponential{Initial: 0, Max: time.Second, Multiplier: 2}
			assert.Equal(t, time.Duration(0), e.Duration(3))
		})

		t.Run("attempt が負なら 0 として扱う", func(t *testing.T) {
			t.Parallel()

			e := Exponential{Initial: 100 * time.Millisecond, Max: time.Second, Multiplier: 2}
			assert.Equal(t, 100*time.Millisecond, e.Duration(-1))
		})
	})
}
