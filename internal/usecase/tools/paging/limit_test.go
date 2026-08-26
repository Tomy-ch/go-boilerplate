package paging

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLimit(t *testing.T) {
	t.Parallel()

	policy := LimitPolicy{Default: 20, Max: 100}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("件数がnilの場合、既定値が使用される", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 20, NewLimit(nil, policy).Value())
		})

		t.Run("件数が0以下の場合、既定値が使用される", func(t *testing.T) {
			t.Parallel()
			zero, negative := 0, -1
			assert.Equal(t, 20, NewLimit(&zero, policy).Value())
			assert.Equal(t, 20, NewLimit(&negative, policy).Value())
		})

		t.Run("件数が範囲内の場合、指定値がそのまま使用される", func(t *testing.T) {
			t.Parallel()
			one, mid, upper := 1, 50, 100
			assert.Equal(t, 1, NewLimit(&one, policy).Value())
			assert.Equal(t, 50, NewLimit(&mid, policy).Value())
			assert.Equal(t, 100, NewLimit(&upper, policy).Value())
		})

		t.Run("件数が上限を超える場合、上限へクランプされる", func(t *testing.T) {
			t.Parallel()
			over := 101
			assert.Equal(t, 100, NewLimit(&over, policy).Value())
		})

		t.Run("ポリシーが異なれば同じ入力でも異なる件数になる", func(t *testing.T) {
			t.Parallel()
			first := 150
			assert.Equal(t, 100, NewLimit(&first, LimitPolicy{Default: 20, Max: 100}).Value())
			assert.Equal(t, 150, NewLimit(&first, LimitPolicy{Default: 50, Max: 200}).Value())
		})

		t.Run("既定値が上限を超えるポリシーでは、既定値も上限へクランプされる", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 10, NewLimit(nil, LimitPolicy{Default: 50, Max: 10}).Value())
		})
	})
}

func TestLimit_Value(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("正規化後の件数を返す", func(t *testing.T) {
			t.Parallel()
			first := 30
			assert.Equal(t, 30, NewLimit(&first, LimitPolicy{Default: 20, Max: 100}).Value())
		})

		t.Run("ゼロ値のLimitは0を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, 0, Limit{}.Value())
		})
	})
}

func TestLimit_Value32(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("正規化後の件数をint32で返す", func(t *testing.T) {
			t.Parallel()
			first := 30
			assert.Equal(t, int32(30), NewLimit(&first, LimitPolicy{Default: 20, Max: 100}).Value32())
		})

		t.Run("int32の上限を超える件数はint32の最大値へクランプされる", func(t *testing.T) {
			t.Parallel()
			first := math.MaxInt32 + 1
			assert.Equal(t, int32(math.MaxInt32), NewLimit(&first, LimitPolicy{Default: 20, Max: math.MaxInt}).Value32())
		})
	})
}
