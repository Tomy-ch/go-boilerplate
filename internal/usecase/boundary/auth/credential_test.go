package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCredential(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("token が空文字でない場合、Credential インスタンスが生成される", func(t *testing.T) {
			t.Parallel()

			expected := &Credential{
				scheme: SchemeBearer,
				token:  "test-token",
			}

			cred, err := NewCredential(SchemeBearer, "test-token")
			require.NoError(t, err)

			assert.Equal(t, expected, cred)
		})

		t.Run("scheme と token は前後の空白が除去される", func(t *testing.T) {
			t.Parallel()

			cred, err := NewCredential("  Bearer  ", "  test-token  ")
			require.NoError(t, err)

			assert.Equal(t, "Bearer", cred.Scheme())
			assert.Equal(t, "test-token", cred.Token())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("token が空文字の場合、ErrTokenMissing エラーになる", func(t *testing.T) {
			t.Parallel()

			cred, err := NewCredential(SchemeBearer, "")

			assert.Nil(t, cred)
			require.ErrorIs(t, err, ErrTokenMissing)
		})

		t.Run("token が空白のみの場合、ErrTokenMissing エラーになる", func(t *testing.T) {
			t.Parallel()

			cred, err := NewCredential(SchemeBearer, "   ")

			assert.Nil(t, cred)
			require.ErrorIs(t, err, ErrTokenMissing)
		})
	})
}

func TestCredential_Scheme(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("コンストラクタで与えた scheme を返す", func(t *testing.T) {
			t.Parallel()
			cred, err := NewCredential(SchemeBearer, "test-token")
			require.NoError(t, err)
			assert.Equal(t, SchemeBearer, cred.Scheme())
		})
	})
}

func TestCredential_Token(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("コンストラクタで与えた token を返す", func(t *testing.T) {
			t.Parallel()
			cred, err := NewCredential(SchemeBearer, "test-token")
			require.NoError(t, err)
			assert.Equal(t, "test-token", cred.Token())
		})
	})
}
