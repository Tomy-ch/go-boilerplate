package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// NewTestLocation は、テスト用のタイムゾーンロケーションを生成します。
func NewTestLocation(t *testing.T) *time.Location {
	t.Helper()
	cfg := MockConfigForTest(t)
	osCfg := NewOperationSystemConfig(cfg)

	loc, err := NewTimeLocation(osCfg)
	require.NoError(t, err)
	return loc
}

// EnsureRepoRootAndEnv は、go.mod のあるリポジトリルートへ作業ディレクトリを移動し（見つかった場合）、
// 併せてテスト用の ENV を t.Setenv で設定します。作業ディレクトリの復元は t.Chdir が自動で行います。
func EnsureRepoRootAndEnv(t *testing.T, env string) {
	t.Helper()

	orig, err := os.Getwd()
	require.NoError(t, err)

	// find repo root by locating go.mod upwards
	p := orig
	for {
		if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
			t.Chdir(p)
			break
		}
		parent := filepath.Dir(p)
		if parent == p {
			break
		}
		p = parent
	}

	t.Setenv(envKey, env)
}
