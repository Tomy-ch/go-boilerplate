package user

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRawPassword(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Run("有効なパスワードの場合、RawPasswordが生成できる", func(t *testing.T) {
			t.Parallel()

			expected := "validPassword123"

			actual, err := NewRawPassword(expected)
			require.NoError(t, err)
			assert.Equal(t, expected, actual.Value())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("パスワードがMinRawPasswordLength未満の場合、エラーになる", func(t *testing.T) {
			t.Parallel()

			invalidPassword := strings.Repeat("a", MinRawPasswordLength-1)

			actual, err := NewRawPassword(invalidPassword)
			assert.Empty(t, actual)
			require.ErrorIs(t, err, ErrInvalidRawPassword)
		})

		t.Run("パスワードがMaxRawPasswordLengthを超える場合、エラーになる", func(t *testing.T) {
			t.Parallel()

			invalidPassword := strings.Repeat("a", MaxRawPasswordLength+1)

			actual, err := NewRawPassword(invalidPassword)
			assert.Empty(t, actual)
			require.ErrorIs(t, err, ErrInvalidRawPassword)
		})
	})
}
