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

	cfg := MockConfigForTest(t)
	osCfg := NewOperationSystemConfig(cfg)

	expected, err := time.LoadLocation(osCfg.TimeZone())
	require.NoError(t, err)

	actual := NewTestLocation(t)
	assert.Equal(t, expected, actual)
}

func TestEnsureRepoRootAndEnv_ChangesDirAndRestores(t *testing.T) {
	// 親のカレントディレクトリを保持
	orig, err := os.Getwd()
	require.NoError(t, err)

	// EnsureRepoRootAndEnv が go.mod のあるリポジトリルートへ移動し復元すること、
	// および go.mod の無い一時ディレクトリでは cwd を変えないことを確認する。
	prevEnv := os.Getenv(envKey)

	t.Run("go.mod を持つリポジトリルートに移動し ENV を設定する", func(t *testing.T) {
		// リポジトリルートを探す
		repoRoot := ""
		p := orig
		for {
			if _, inErr := os.Stat(filepath.Join(p, "go.mod")); inErr == nil {
				repoRoot = p
				break
			}
			parent := filepath.Dir(p)
			if parent == p {
				break
			}
			p = parent
		}
		require.NotEmpty(t, repoRoot)

		EnsureRepoRootAndEnv(t, TestingEnvValue)

		cwd, inErr := os.Getwd()
		require.NoError(t, inErr)
		// 呼び出し先のカレントディレクトリが go.mod を持つルートであること
		assert.FileExists(t, filepath.Join(cwd, "go.mod"))

		// ENV が設定されていること
		assert.Equal(t, TestingEnvValue, os.Getenv(envKey))
	})

	t.Run("go.mod が見つからない場合は cwd を変更せず ENV を設定する", func(t *testing.T) {
		// 前提: t.TempDir() の上位に go.mod が存在しない（探索が終端まで到達する）こと。
		tmp := t.TempDir()
		t.Chdir(tmp)

		EnsureRepoRootAndEnv(t, TestingEnvValue)

		cwd, inErr := os.Getwd()
		require.NoError(t, inErr)
		// go.mod が見つからないため cwd は変更されない
		assert.Equal(t, tmp, cwd)

		// ただし ENV は設定される
		assert.Equal(t, TestingEnvValue, os.Getenv(envKey))
	})

	// 復元されていることを確認
	cwdAfter, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, orig, cwdAfter)
	assert.Equal(t, prevEnv, os.Getenv(envKey))
}
