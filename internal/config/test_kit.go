package config

import (
	"fmt"
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

	setWorktreeDBName(t, env)
}

// setWorktreeDBName は、DB スロットプール利用時に対象 env の DB_NAME を自 worktree のデータベースへ
// 上書きする。local は DB_NAME_LOCAL（wt<N>_local）、ci/test は DB_NAME_TEST（wt<N>_test）。deploy 系
// env では本番 DB を誤指しないよう無視する（IsLocalClassEnv）。未使用時は .env.<env> の値のまま。
func setWorktreeDBName(t *testing.T, env string) {
	t.Helper()
	switch {
	case env == EnvLocal && IsLocalClassEnv(env):
		if v := os.Getenv("DB_NAME_LOCAL"); v != "" {
			t.Setenv("DB_NAME", v)
		}
	case IsLocalClassEnv(env):
		t.Setenv("DB_NAME", testDBName())
	case os.Getenv("DB_NAME_TEST") != "" || os.Getenv("DB_NAME_LOCAL") != "":
		fmt.Fprintf(os.Stderr,
			"[config] 警告: env=%q は local/test 系でないため、DB スロットプールの向き先変更を無視します\n", env)
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
