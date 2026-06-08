package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_WithEnvSet(t *testing.T) {
	t.Run("ENV が設定されている場合、Load はエラーなく成功する", func(t *testing.T) {
		EnsureRepoRootAndEnv(t, TestingEnvValue)

		err := Load()
		require.NoError(t, err)
	})

	t.Run("env/.env は存在するが ENV が空のため ErrEnvNotResolved を返す", func(t *testing.T) {
		EnsureRepoRootAndEnv(t, TestingEnvValue)
		t.Setenv(envKey, "")

		err := Load()
		require.ErrorIs(t, err, ErrEnvNotResolved)
	})

	t.Run("デフォルトの .env ファイルが存在しない場合、Load はエラーを返す", func(t *testing.T) {
		tmp := t.TempDir()
		t.Chdir(tmp)
		t.Setenv(envKey, "")

		err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "env/.env")
	})
}
