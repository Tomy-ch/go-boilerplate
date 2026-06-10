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

func TestEnsureRepoRootAndEnv(t *testing.T) {
	// t.Setenv / t.Chdir でプロセス状態を書き換えるため Parallel は使用しない。
	orig, err := os.Getwd()
	require.NoError(t, err)
	prevEnv := os.Getenv(envKey)

	t.Run("正常系", func(t *testing.T) {
		t.Run("go.mod が見つかる場合、リポジトリルートに移動し ENV を設定する", func(t *testing.T) {
			EnsureRepoRootAndEnv(t, TestingEnvValue)

			cwd, inErr := os.Getwd()
			require.NoError(t, inErr)
			assert.FileExists(t, filepath.Join(cwd, "go.mod"))
			assert.Equal(t, TestingEnvValue, os.Getenv(envKey))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("go.mod が見つからない場合、cwd は変更されず ENV のみ設定される", func(t *testing.T) {
			tmp := t.TempDir()
			t.Chdir(tmp)

			EnsureRepoRootAndEnv(t, TestingEnvValue)

			cwd, inErr := os.Getwd()
			require.NoError(t, inErr)
			assert.Equal(t, tmp, cwd)
			assert.Equal(t, TestingEnvValue, os.Getenv(envKey))
		})
	})

	// サブテスト終了時に t.Setenv / t.Chdir がクリーンアップされ、副作用が漏れないこと。
	cwdAfter, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, orig, cwdAfter)
	assert.Equal(t, prevEnv, os.Getenv(envKey))
}
