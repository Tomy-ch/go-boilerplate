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

func TestIntToInt16(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("0は0を返す", func(t *testing.T) {
			t.Parallel()
			result, err := IntToInt16(0)
			require.NoError(t, err)
			assert.Equal(t, int16(0), result)
		})

		t.Run("MaxInt16と等しい値はMaxInt16を返す", func(t *testing.T) {
			t.Parallel()
			result, err := IntToInt16(math.MaxInt16)
			require.NoError(t, err)
			assert.Equal(t, int16(math.MaxInt16), result)
		})

		t.Run("MinInt16と等しい値はMinInt16を返す", func(t *testing.T) {
			t.Parallel()
			result, err := IntToInt16(math.MinInt16)
			require.NoError(t, err)
			assert.Equal(t, int16(math.MinInt16), result)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("MaxInt16+1はオーバーフローエラーを返す", func(t *testing.T) {
			t.Parallel()
			result, err := IntToInt16(math.MaxInt16 + 1)
			require.ErrorIs(t, err, ErrOverflow)
			assert.Zero(t, result)
		})

		t.Run("MinInt16-1はオーバーフローエラーを返す", func(t *testing.T) {
			t.Parallel()
			result, err := IntToInt16(math.MinInt16 - 1)
			require.ErrorIs(t, err, ErrOverflow)
			assert.Zero(t, result)
		})
	})
}

func TestIntPtrToInt32Ptr(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilはnilを返す", func(t *testing.T) {
			t.Parallel()
			result, err := IntPtrToInt32Ptr(nil)
			require.NoError(t, err)
			assert.Nil(t, result)
		})

		t.Run("範囲内の値は値を保持したポインタを返す", func(t *testing.T) {
			t.Parallel()
			v := 42
			result, err := IntPtrToInt32Ptr(&v)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, int32(42), *result)
		})

		t.Run("MaxInt32とMinInt32も値を保持したポインタを返す", func(t *testing.T) {
			t.Parallel()
			maxValue, minValue := math.MaxInt32, math.MinInt32

			maxResult, err := IntPtrToInt32Ptr(&maxValue)
			require.NoError(t, err)
			require.NotNil(t, maxResult)
			assert.Equal(t, int32(math.MaxInt32), *maxResult)

			minResult, err := IntPtrToInt32Ptr(&minValue)
			require.NoError(t, err)
			require.NotNil(t, minResult)
			assert.Equal(t, int32(math.MinInt32), *minResult)
		})

		t.Run("返すポインタは引数とは別の領域を指す", func(t *testing.T) {
			t.Parallel()
			v := 7
			result, err := IntPtrToInt32Ptr(&v)
			require.NoError(t, err)
			require.NotNil(t, result)

			v = 99
			assert.Equal(t, int32(7), *result)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("MaxInt32+1はオーバーフローエラーを返す", func(t *testing.T) {
			t.Parallel()
			v := math.MaxInt32 + 1
			result, err := IntPtrToInt32Ptr(&v)
			require.ErrorIs(t, err, ErrOverflow)
			assert.Nil(t, result)
		})

		t.Run("MinInt32-1はオーバーフローエラーを返す", func(t *testing.T) {
			t.Parallel()
			v := math.MinInt32 - 1
			result, err := IntPtrToInt32Ptr(&v)
			require.ErrorIs(t, err, ErrOverflow)
			assert.Nil(t, result)
		})
	})
}
