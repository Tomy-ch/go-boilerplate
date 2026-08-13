package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const requireShellcheckEnv = "REQUIRE_SHELLCHECK"

const cleanScript = "#!/bin/sh\nset -eu\necho hello\n"

// SC2086: 引用符の無い展開。方言に依存せず必ず指摘される。
const dirtyScript = "#!/bin/sh\nset -eu\nx=$1\necho $x\n"

func Test_run(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指摘の無いスクリプトだけなら成功する", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)

			root := writeScripts(t, map[string]string{"ok.sh": cleanScript})

			require.NoError(t, run(t.Context(), rootAt(root), exec.LookPath))
		})

		t.Run("シェルスクリプトが 1 つも無くても成功する", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)

			root := writeScripts(t, map[string]string{"README.md": "not a script\n"})

			require.NoError(t, run(t.Context(), rootAt(root), exec.LookPath))
		})

		t.Run("除外ディレクトリ配下は検査しない", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)

			root := writeScripts(t, map[string]string{"vendor/bad.sh": dirtyScript})

			require.NoError(t, run(t.Context(), rootAt(root), exec.LookPath))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指摘のあるスクリプトを検出する", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)

			root := writeScripts(t, map[string]string{"bad.sh": dirtyScript})

			require.ErrorIs(t, run(t.Context(), rootAt(root), exec.LookPath), errFindings)
		})

		t.Run("shellcheck が無ければ実行せずに報告する", func(t *testing.T) {
			t.Parallel()

			missing := func(string) (string, error) { return "", os.ErrNotExist }

			require.ErrorIs(t, run(t.Context(), rootAt(t.TempDir()), missing), errShellcheckMissing)
		})

		t.Run("基点ディレクトリを解決できなければ報告する", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)

			failing := func() (string, error) { return "", os.ErrPermission }

			require.ErrorIs(t, run(t.Context(), failing, exec.LookPath), os.ErrPermission)
		})

		t.Run("shellcheck の起動自体に失敗したら報告する", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)

			root := writeScripts(t, map[string]string{"ok.sh": cleanScript})

			require.ErrorIs(t, run(canceledContext(t), rootAt(root), exec.LookPath), errShellcheck)
		})
	})
}

func Test_shellScripts(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("リポジトリ相対パスを昇順で返す", func(t *testing.T) {
			t.Parallel()

			root := writeScripts(t, map[string]string{
				"b.sh":        cleanScript,
				"a/nested.sh": cleanScript,
				"note.txt":    "x\n",
			})

			scripts, err := shellScripts(root)

			require.NoError(t, err)
			assert.Equal(t, []string{"a/nested.sh", "b.sh"}, scripts)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("走査できない基点なら報告する", func(t *testing.T) {
			t.Parallel()

			_, err := shellScripts(filepath.Join(t.TempDir(), "missing"))

			require.Error(t, err)
		})
	})
}

func Test_prefixFindings(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("パスをリポジトリ相対へ揃える", func(t *testing.T) {
			t.Parallel()

			out := "/abs/path/bad.sh:4:6: note: Double quote [SC2086]\n"

			assert.Equal(t, []string{"a/bad.sh:4:6: note: Double quote [SC2086]"}, prefixFindings("a/bad.sh", out))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空の出力を指摘にしない", func(t *testing.T) {
			t.Parallel()

			assert.Nil(t, prefixFindings("a/bad.sh", "  \n"))
		})

		t.Run("区切りを持たない行を落とす", func(t *testing.T) {
			t.Parallel()

			assert.Nil(t, prefixFindings("a/bad.sh", "malformed line\n"))
		})
	})
}

// writeScripts は相対パスと内容の対からテスト用のツリーを作り、その基点を返します。
func writeScripts(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
		require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	}

	return root
}

// rootAt は固定の基点を返す wd 相当の関数を作ります。
func rootAt(root string) func() (string, error) {
	return func() (string, error) { return root, nil }
}

func requireShellcheck(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath(shellcheckBin); err == nil {
		return
	}
	if os.Getenv(requireShellcheckEnv) != "" {
		t.Fatalf("shellcheck が PATH にありません（%s 指定時は skip しません）", requireShellcheckEnv)
	}
	t.Skip("shellcheck が PATH にありません")
}

func canceledContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	return ctx
}

func Test_runShellcheck(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指摘の無い内容には何も返さない", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)

			out, err := runShellcheck(t.Context(), cleanScript)

			require.NoError(t, err)
			assert.Empty(t, strings.TrimSpace(out))
		})

		t.Run("指摘のある内容を gcc 形式で返す", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)

			out, err := runShellcheck(t.Context(), dirtyScript)

			require.NoError(t, err)
			assert.Contains(t, out, "SC2086")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("起動できなければ報告する", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)

			_, err := runShellcheck(canceledContext(t), cleanScript)

			require.ErrorIs(t, err, errShellcheck)
		})
	})
}
