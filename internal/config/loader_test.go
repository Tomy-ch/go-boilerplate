package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Run("正常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv/t.Chdir使用のため並列化不可
		t.Run("ENV が設定されている場合、Load はエラーなく成功する", func(t *testing.T) { //nolint:paralleltest // t.Setenv/t.Chdir使用のため並列化不可
			EnsureRepoRootAndEnv(t, TestingEnvValue)

			err := Load()
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("デフォルトの .env ファイルが存在しない場合、ErrFailedToLoadDefaultEnvFile を返す", func(t *testing.T) {
			tmp := t.TempDir()
			t.Chdir(tmp)
			t.Setenv(envKey, "")

			err := Load()
			require.ErrorIs(t, err, ErrFailedToLoadDefaultEnvFile)
		})

		t.Run("env/.env は存在するが ENV が空のため ErrEnvNotResolved を返す", func(t *testing.T) {
			EnsureRepoRootAndEnv(t, TestingEnvValue)
			t.Setenv(envKey, "")

			err := Load()
			require.ErrorIs(t, err, ErrEnvNotResolved)
		})

		t.Run("ENV に対応する .env.<env> が存在しない場合、ErrFailedToLoadEnvFile を返す", func(t *testing.T) {
			EnsureRepoRootAndEnv(t, TestingEnvValue)
			t.Setenv(envKey, "nonexistent_env")

			err := Load()
			require.ErrorIs(t, err, ErrFailedToLoadEnvFile)
		})
	})
}
