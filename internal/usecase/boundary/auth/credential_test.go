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

		t.Run("accessToken が空文字でない場合、Credential インスタンスが生成される", func(t *testing.T) {
			t.Parallel()

			expected := &Credential{
				accessToken: "test-access-token",
			}

			cred, err := NewCredential("test-access-token")
			require.NoError(t, err)

			assert.Equal(t, expected, cred)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("accessToken が空文字の場合、ErrTokenMissing エラーになる", func(t *testing.T) {
			t.Parallel()

			cred, err := NewCredential("")

			assert.Nil(t, cred)
			require.ErrorIs(t, err, ErrTokenMissing)
		})

		t.Run("accessToken が空白のみの場合、ErrTokenMissing エラーになる", func(t *testing.T) {
			t.Parallel()

			cred, err := NewCredential("   ")

			assert.Nil(t, cred)
			require.ErrorIs(t, err, ErrTokenMissing)
		})
	})
}

func TestCredential_AccessToken(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("コンストラクタで与えたtokenを返す", func(t *testing.T) {
			t.Parallel()
			cred, err := NewCredential("test-access-token")
			require.NoError(t, err)
			assert.Equal(t, "test-access-token", cred.AccessToken())
		})
	})
}
