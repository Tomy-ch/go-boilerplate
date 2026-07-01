package safecast

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUintToInt(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("0は0を返す", func(t *testing.T) {
			t.Parallel()
			result, err := UintToInt(0)
			require.NoError(t, err)
			assert.Equal(t, 0, result)
		})

		t.Run("MaxIntと等しい値はMaxIntを返す", func(t *testing.T) {
			t.Parallel()
			result, err := UintToInt(uint(math.MaxInt))
			require.NoError(t, err)
			assert.Equal(t, math.MaxInt, result)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("MaxInt+1はオーバーフローエラーを返す", func(t *testing.T) {
			t.Parallel()
			result, err := UintToInt(math.MaxInt + 1)
			require.ErrorIs(t, err, ErrOverflow)
			assert.Equal(t, 0, result)
		})

		t.Run("MaxUintはオーバーフローエラーを返す", func(t *testing.T) {
			t.Parallel()
			result, err := UintToInt(math.MaxUint)
			require.ErrorIs(t, err, ErrOverflow)
			assert.Equal(t, 0, result)
		})
	})
}
