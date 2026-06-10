package security

import (
	"strings"
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestBcryptHasher_Hash(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	secCfg := config.NewSecurityConfig(cfg)

	hasher := NewBcryptHasher(secCfg)

	hash, err := hasher.Hash("password")

	require.NoError(t, err)
	require.NotEmpty(t, hash)
	assert.NotEqual(t, "password", hash)
}

func TestBcryptHasher_Hash_Error(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	secCfg := config.NewSecurityConfig(cfg)

	hasher := NewBcryptHasher(secCfg)

	// bcrypt は72バイト超のパスワードを拒否する。
	hash, err := hasher.Hash(strings.Repeat("a", 73))

	require.ErrorIs(t, err, bcrypt.ErrPasswordTooLong)
	assert.Empty(t, hash)
}

func TestBcryptHasher_Compare_Error(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	secCfg := config.NewSecurityConfig(cfg)

	hasher := NewBcryptHasher(secCfg)

	// 不一致以外の失敗（不正なハッシュ）はエラーとして返る。
	ok, err := hasher.Compare("not-a-bcrypt-hash", "password")

	require.ErrorIs(t, err, bcrypt.ErrHashTooShort)
	assert.False(t, ok)
}

func TestBcryptHasher_Compare(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	secCfg := config.NewSecurityConfig(cfg)

	hasher := NewBcryptHasher(secCfg)

	password := "password"

	hash, err := hasher.Hash(password)
	require.NoError(t, err)

	ok, err := hasher.Compare(hash, password)

	require.NoError(t, err)
	assert.True(t, ok)
}

func TestBcryptHasher_Compare_Mismatch(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	secCfg := config.NewSecurityConfig(cfg)

	hasher := NewBcryptHasher(secCfg)

	hash, err := hasher.Hash("password")
	require.NoError(t, err)

	ok, err := hasher.Compare(hash, "wrong-password")

	require.NoError(t, err)
	assert.False(t, ok)
}
