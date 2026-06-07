package safecast

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUint32ToInt(t *testing.T) {
	t.Run("Minimum value", func(t *testing.T) {
		t.Parallel()
		result, err := UintToInt(0)
		require.NoError(t, err)
		assert.Equal(t, 0, result)
	})

	t.Run("Maximum int value", func(t *testing.T) {
		t.Parallel()
		result, err := UintToInt(uint(math.MaxInt))
		require.NoError(t, err)
		assert.Equal(t, math.MaxInt, result)
	})

	t.Run("Overflow just above max int", func(t *testing.T) {
		t.Parallel()
		result, err := UintToInt(math.MaxInt + 1)
		assert.Equal(t, 0, result)
		require.ErrorIs(t, err, ErrOverflow)
	})

	t.Run("Maximum uint value", func(t *testing.T) {
		t.Parallel()
		result, err := UintToInt(math.MaxUint)
		assert.Equal(t, 0, result)
		require.ErrorIs(t, err, ErrOverflow)
	})
}
