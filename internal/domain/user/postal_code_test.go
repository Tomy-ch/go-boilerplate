package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPostalCode(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("NNN-NNNN 形式の場合、PostalCode が構築され Value が入力値を返す", func(t *testing.T) {
			t.Parallel()
			actual, err := NewPostalCode("150-0001")
			require.NoError(t, err)
			assert.Equal(t, "150-0001", actual.Value())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空文字の場合、ErrInvalidPostalCode を返す", func(t *testing.T) {
			t.Parallel()
			_, err := NewPostalCode("")
			require.ErrorIs(t, err, ErrInvalidPostalCode)
		})

		t.Run("ハイフンが無い場合、ErrInvalidPostalCode を返す", func(t *testing.T) {
			t.Parallel()
			_, err := NewPostalCode("1500001")
			require.ErrorIs(t, err, ErrInvalidPostalCode)
		})

		t.Run("桁数が不足する場合、ErrInvalidPostalCode を返す", func(t *testing.T) {
			t.Parallel()
			_, err := NewPostalCode("15-0001")
			require.ErrorIs(t, err, ErrInvalidPostalCode)
		})

		t.Run("数字以外を含む場合、ErrInvalidPostalCode を返す", func(t *testing.T) {
			t.Parallel()
			_, err := NewPostalCode("abc-defg")
			require.ErrorIs(t, err, ErrInvalidPostalCode)
		})
	})
}
