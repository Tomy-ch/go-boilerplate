package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

// NewTestLocation は、テスト用のタイムゾーンロケーションを生成します。
func NewTestLocation(t *testing.T) *time.Location {
	t.Helper()
	cfg := MockConfigForTest(t)
	osCfg := NewOperatingSystemConfig(cfg)

	loc, err := NewTimeLocation(osCfg)
	require.NoError(t, err)
	return loc
}

// EnsureRepoRootAndEnv は、go.mod のあるリポジトリルートへ作業ディレクトリを移動し、
// env/.env.<env> の各値を t.Setenv で設定します。埋め込み（local）より対象環境の値が優先されます。
// 作業ディレクトリと環境変数の復元は t.Chdir / t.Setenv が自動で行います。
func EnsureRepoRootAndEnv(t *testing.T, env string) {
	t.Helper()

	root := repoRoot(t)
	t.Chdir(root)

	kv, err := godotenv.Read(filepath.Join(root, "env", ".env."+env))
	require.NoError(t, err)
	for k, v := range kv {
		t.Setenv(k, v)
	}
}

// repoRoot は、go.mod を上方向に探索してリポジトリルートを返します。
func repoRoot(t *testing.T) string {
	t.Helper()

	p, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
			return p
		}
		parent := filepath.Dir(p)
		require.NotEqual(t, parent, p, "リポジトリルート（go.mod）が見つかりません")
		p = parent
	}
}
