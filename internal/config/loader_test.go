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

	t.Run("ENV が設定されていない場合、Load はエラーを返す", func(t *testing.T) {
		EnsureRepoRootAndEnv(t, TestingEnvValue)
		t.Setenv(envKey, "")

		err := Load()
		require.Error(t, err)
		// リポジトリファイルによっては、エラーが env/.env または env/.env.<空ファイル> を報告する場合があります
		assert.Contains(t, err.Error(), "env/.env")
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
