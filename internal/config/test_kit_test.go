package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestLocation(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("MockConfig のタイムゾーンに対応するロケーションが返ること", func(t *testing.T) {
			t.Parallel()

			cfg := MockConfigForTest(t)
			osCfg := NewOperatingSystemConfig(cfg)

			expected, err := time.LoadLocation(osCfg.TimeZone())
			require.NoError(t, err)

			actual := NewTestLocation(t)
			assert.Equal(t, expected, actual)
		})
	})
}

func TestEnsureRepoRootAndEnv(t *testing.T) { //nolint:paralleltest // t.Setenv/t.Chdir使用のため並列化不可
	// t.Setenv / t.Chdir でプロセス状態を書き換えるため Parallel は使用しない。
	orig, err := os.Getwd()
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv/t.Chdir使用のため並列化不可
		t.Run("リポジトリルートへ移動し対象 env の値を環境変数へ設定する", func(t *testing.T) { //nolint:paralleltest // t.Setenv/t.Chdir使用のため並列化不可
			EnsureRepoRootAndEnv(t, TestingEnvValue)

			cwd, inErr := os.Getwd()
			require.NoError(t, inErr)
			assert.FileExists(t, filepath.Join(cwd, "go.mod"))
			// env/.env.<env> の値が環境変数へ反映される。
			assert.Equal(t, TestingEnvValue, os.Getenv("APP_ENV"))
		})
	})

	// サブテスト終了時に t.Chdir がクリーンアップされ、cwd が復元されること。
	cwdAfter, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, orig, cwdAfter)
}
