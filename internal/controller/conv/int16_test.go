package conv

import (
	"math"
	"testing"

	"go-boilerplate/pkg/safecast"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInt16sPtr(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定された値を並び順どおりint16のスライスへ変換する", func(t *testing.T) {
			t.Parallel()

			actual, err := Int16sPtr(&[]int32{3, 1, 2})

			require.NoError(t, err)
			assert.Equal(t, []int16{3, 1, 2}, actual)
		})

		t.Run("nilは絞り込みなしとしてnilを返す", func(t *testing.T) {
			t.Parallel()

			actual, err := Int16sPtr(nil)

			require.NoError(t, err)
			assert.Nil(t, actual)
		})

		t.Run("空配列も絞り込みなしとしてnilを返す", func(t *testing.T) {
			t.Parallel()

			actual, err := Int16sPtr(&[]int32{})

			require.NoError(t, err)
			assert.Nil(t, actual)
		})

		t.Run("int16の境界値をそのまま保持する", func(t *testing.T) {
			t.Parallel()

			actual, err := Int16sPtr(&[]int32{math.MinInt16, math.MaxInt16})

			require.NoError(t, err)
			assert.Equal(t, []int16{math.MinInt16, math.MaxInt16}, actual)
		})

		t.Run("変換後のスライスは引数と独立していて呼び出し側の書き換えの影響を受けない", func(t *testing.T) {
			t.Parallel()

			src := []int32{1, 2}
			actual, err := Int16sPtr(&src)
			require.NoError(t, err)
			src[0] = 9

			assert.Equal(t, []int16{1, 2}, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("int16の上限を超える値がある場合、ErrOverflowを返す", func(t *testing.T) {
			t.Parallel()

			actual, err := Int16sPtr(&[]int32{1, math.MaxInt16 + 1})

			require.ErrorIs(t, err, safecast.ErrOverflow)
			assert.Nil(t, actual)
		})

		t.Run("int16の下限を下回る値がある場合、ErrOverflowを返す", func(t *testing.T) {
			t.Parallel()

			actual, err := Int16sPtr(&[]int32{math.MinInt16 - 1})

			require.ErrorIs(t, err, safecast.ErrOverflow)
			assert.Nil(t, actual)
		})
	})
}
