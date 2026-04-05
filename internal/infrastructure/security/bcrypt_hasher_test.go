package security

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/require"
)

func TestBcryptHasher_Hash(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	secCfg := config.NewSecurityConfig(cfg)

	hasher := NewBcryptHasher(secCfg)

	hash, err := hasher.Hash("password")

	require.NoError(t, err)
	require.NotEmpty(t, hash)
	require.NotEqual(t, "password", hash)
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
	require.True(t, ok)
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
	require.False(t, ok)
}
