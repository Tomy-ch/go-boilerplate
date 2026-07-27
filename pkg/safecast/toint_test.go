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

func TestIntToInt32(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("0は0を返す", func(t *testing.T) {
			t.Parallel()
			result, err := IntToInt32(0)
			require.NoError(t, err)
			assert.Equal(t, int32(0), result)
		})

		t.Run("MaxInt32と等しい値はMaxInt32を返す", func(t *testing.T) {
			t.Parallel()
			result, err := IntToInt32(math.MaxInt32)
			require.NoError(t, err)
			assert.Equal(t, int32(math.MaxInt32), result)
		})

		t.Run("MinInt32と等しい値はMinInt32を返す", func(t *testing.T) {
			t.Parallel()
			result, err := IntToInt32(math.MinInt32)
			require.NoError(t, err)
			assert.Equal(t, int32(math.MinInt32), result)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("MaxInt32+1はオーバーフローエラーを返す", func(t *testing.T) {
			t.Parallel()
			result, err := IntToInt32(math.MaxInt32 + 1)
			require.ErrorIs(t, err, ErrOverflow)
			assert.Zero(t, result)
		})

		t.Run("MinInt32-1はオーバーフローエラーを返す", func(t *testing.T) {
			t.Parallel()
			result, err := IntToInt32(math.MinInt32 - 1)
			require.ErrorIs(t, err, ErrOverflow)
			assert.Zero(t, result)
		})
	})
}
