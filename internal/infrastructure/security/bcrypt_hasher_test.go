package security

import (
	"strings"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBcryptHasher(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定を渡すと非nilのHasherを返す", func(t *testing.T) {
			t.Parallel()

			hasher := NewBcryptHasher(config.NewSecurityConfig(config.MockConfigForTest(t)))

			assert.NotNil(t, hasher)
		})
	})
}

func TestBcryptHasher_Hash(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ハッシュ化に成功し平文とは異なる値を返す", func(t *testing.T) {
			t.Parallel()
			hasher := NewBcryptHasher(config.NewSecurityConfig(config.MockConfigForTest(t)))

			hash, err := hasher.Hash("password")
			require.NoError(t, err)
			assert.NotEmpty(t, hash)
			assert.NotEqual(t, "password", hash)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("72バイト超のパスワードはErrInternalへ変換される", func(t *testing.T) {
			t.Parallel()
			hasher := NewBcryptHasher(config.NewSecurityConfig(config.MockConfigForTest(t)))

			hash, err := hasher.Hash(strings.Repeat("a", 73))
			require.ErrorIs(t, err, apperror.ErrInternal)
			require.ErrorContains(t, err, "bcrypt hash failed")
			assert.Empty(t, hash)
		})
	})
}

func TestBcryptHasher_Compare(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ハッシュとパスワードが一致する場合trueを返す", func(t *testing.T) {
			t.Parallel()
			hasher := NewBcryptHasher(config.NewSecurityConfig(config.MockConfigForTest(t)))

			hash, err := hasher.Hash("password")
			require.NoError(t, err)

			ok, err := hasher.Compare(hash, "password")
			require.NoError(t, err)
			assert.True(t, ok)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("パスワードが不一致の場合falseとnilを返す", func(t *testing.T) {
			t.Parallel()
			hasher := NewBcryptHasher(config.NewSecurityConfig(config.MockConfigForTest(t)))

			hash, err := hasher.Hash("password")
			require.NoError(t, err)

			ok, err := hasher.Compare(hash, "wrong-password")
			require.NoError(t, err)
			assert.False(t, ok)
		})

		t.Run("不正なハッシュ形式の場合ErrInternalへ変換される", func(t *testing.T) {
			t.Parallel()
			hasher := NewBcryptHasher(config.NewSecurityConfig(config.MockConfigForTest(t)))

			ok, err := hasher.Compare("not-a-bcrypt-hash", "password")
			require.ErrorIs(t, err, apperror.ErrInternal)
			require.ErrorContains(t, err, "bcrypt compare failed")
			assert.False(t, ok)
		})
	})
}
