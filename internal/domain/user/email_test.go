package user

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmail(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な形式の場合、Email が構築され Value が入力値を返す", func(t *testing.T) {
			t.Parallel()
			actual, err := NewEmail("john.doe@example.com")
			require.NoError(t, err)
			assert.Equal(t, "john.doe@example.com", actual.Value())
		})

		t.Run("最大文字数ちょうどの有効な形式の場合、Email が構築される", func(t *testing.T) {
			t.Parallel()
			local := strings.Repeat("a", maxEmailLength-len("@example.com"))
			actual, err := NewEmail(local + "@example.com")
			require.NoError(t, err)
			assert.Equal(t, local+"@example.com", actual.Value())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空文字の場合、ErrInvalidEmail を返す", func(t *testing.T) {
			t.Parallel()
			_, err := NewEmail("")
			require.ErrorIs(t, err, ErrInvalidEmail)
		})

		t.Run("最大文字数を超える場合、ErrInvalidEmail を返す", func(t *testing.T) {
			t.Parallel()
			local := strings.Repeat("a", maxEmailLength-len("@example.com")+1)
			_, err := NewEmail(local + "@example.com")
			require.ErrorIs(t, err, ErrInvalidEmail)
		})

		t.Run("アットマークが無い場合、ErrInvalidEmail を返す", func(t *testing.T) {
			t.Parallel()
			_, err := NewEmail("not-an-email")
			require.ErrorIs(t, err, ErrInvalidEmail)
		})

		t.Run("ドメインにドットが無い場合、ErrInvalidEmail を返す", func(t *testing.T) {
			t.Parallel()
			_, err := NewEmail("user@localhost")
			require.ErrorIs(t, err, ErrInvalidEmail)
		})

		t.Run("空白を含む場合、ErrInvalidEmail を返す", func(t *testing.T) {
			t.Parallel()
			_, err := NewEmail("user name@example.com")
			require.ErrorIs(t, err, ErrInvalidEmail)
		})
	})
}

func TestEmail_Value(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
