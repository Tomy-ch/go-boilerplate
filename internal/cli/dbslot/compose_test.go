package dbslot

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubDocker は、PATH 先頭に引数と COMPOSE_PROJECT_NAME を記録するダミー docker を仕込み、記録ファイルの
// パスを返す。実 docker を呼ばずに ExecCompose の実装（コマンド構築・環境変数注入）を検証する。
func stubDocker(t *testing.T, exit int) string {
	t.Helper()
	dir := t.TempDir()
	out := filepath.Join(dir, "out")
	script := "#!/bin/sh\necho \"proj=$COMPOSE_PROJECT_NAME args=$*\" >> " + out + "\nexit " + strconv.Itoa(exit) + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docker"), []byte(script), 0o755)) //nolint:gosec // 実行可能スタブ
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return out
}

func TestExecCompose_UpSharedDB(t *testing.T) { //nolint:paralleltest // stubDocker が t.Setenv を使うため並列化不可
	out := stubDocker(t, 0)

	t.Run("正常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
		t.Run("固定プロジェクトで database profile を up --wait する", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			require.NoError(t, ExecCompose{}.UpSharedDB(context.Background(), "myproj"))

			b, err := os.ReadFile(out) //nolint:gosec // テスト内の固定パス
			require.NoError(t, err)
			assert.Contains(t, string(b), "proj=myproj")
			assert.Contains(t, string(b), "compose --profile database up -d --wait database")
		})
	})
}

func TestExecCompose_DownServe(t *testing.T) { //nolint:paralleltest // stubDocker が t.Setenv を使うため並列化不可
	t.Run("正常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
		t.Run("serve プロジェクトを attach override 付きで down する", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			out := stubDocker(t, 0)

			require.NoError(t, ExecCompose{}.DownServe(context.Background(), "gobp-wt-1"))

			b, err := os.ReadFile(out) //nolint:gosec // テスト内の固定パス
			require.NoError(t, err)
			assert.Contains(t, string(b), "proj=gobp-wt-1")
			assert.Contains(t, string(b), "-f docker-compose.yaml -f docker-compose.attach.yaml down")
		})

		t.Run("down が失敗してもエラーにしない（冪等）", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			stubDocker(t, 1)

			require.NoError(t, ExecCompose{}.DownServe(context.Background(), "gobp-wt-1"))
		})
	})
}

func TestExecCompose_run(t *testing.T) { //nolint:paralleltest // stubDocker が t.Setenv を使うため並列化不可
	t.Run("異常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
		t.Run("docker が非 0 終了ならエラーを返す", func(t *testing.T) { //nolint:paralleltest // t.Setenv 使用
			stubDocker(t, 1)

			err := ExecCompose{}.UpSharedDB(context.Background(), "p")
			require.Error(t, err)
		})
	})
}
