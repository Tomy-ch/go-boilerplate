package ghfiles_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/scripts/lib/ghfiles"
)

// writeFile は親ディレクトリごとファイルを作る。
func writeFile(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte("jobs:\n"), 0o600))
}

func Test_Collect(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("workflow 定義を拡張子の違いごと集める", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, filepath.Join(root, ".github", "workflows", "ci.yaml"))
			writeFile(t, filepath.Join(root, ".github", "workflows", "lint.yml"))

			got, err := ghfiles.Collect(root)
			require.NoError(t, err)
			assert.Equal(t, []string{
				filepath.Join(root, ".github", "workflows", "ci.yaml"),
				filepath.Join(root, ".github", "workflows", "lint.yml"),
			}, got)
		})

		t.Run("入れ子に置かれた composite action 定義まで再帰的に集める", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, filepath.Join(root, ".github", "actions", "setup", "action.yml"))
			writeFile(t, filepath.Join(root, ".github", "actions", "g", "nested", "action.yaml"))

			got, err := ghfiles.Collect(root)
			require.NoError(t, err)
			assert.Equal(t, []string{
				filepath.Join(root, ".github", "actions", "g", "nested", "action.yaml"),
				filepath.Join(root, ".github", "actions", "setup", "action.yml"),
			}, got)
		})

		t.Run("workflow と composite action を混ぜてもパス順に整列する", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, filepath.Join(root, ".github", "workflows", "ci.yaml"))
			writeFile(t, filepath.Join(root, ".github", "actions", "setup", "action.yml"))

			got, err := ghfiles.Collect(root)
			require.NoError(t, err)
			assert.Equal(t, []string{
				filepath.Join(root, ".github", "actions", "setup", "action.yml"),
				filepath.Join(root, ".github", "workflows", "ci.yaml"),
			}, got)
		})

		t.Run(".github が無いリポジトリでは空を返す", func(t *testing.T) {
			t.Parallel()

			got, err := ghfiles.Collect(t.TempDir())
			require.NoError(t, err)
			assert.Empty(t, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("workflow ディレクトリの YAML 以外は集めない", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, filepath.Join(root, ".github", "workflows", "README.md"))

			got, err := ghfiles.Collect(root)
			require.NoError(t, err)
			assert.Empty(t, got)
		})

		t.Run("composite action ディレクトリの定義ファイル以外は集めない", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, filepath.Join(root, ".github", "actions", "setup", "README.md"))
			writeFile(t, filepath.Join(root, ".github", "actions", "setup", "entrypoint.sh"))

			got, err := ghfiles.Collect(root)
			require.NoError(t, err)
			assert.Empty(t, got)
		})

		t.Run("不在以外の走査失敗は握り潰さず返す", func(t *testing.T) {
			t.Parallel()
			if os.Geteuid() == 0 {
				t.Skip("root では読み取り権限の剥奪が効かない")
			}
			root := t.TempDir()
			actions := filepath.Join(root, ".github", "actions")
			writeFile(t, filepath.Join(actions, "setup", "action.yml"))
			require.NoError(t, os.Chmod(actions, 0o000))
			// TempDir の後片付けが子を unlink できるよう書き込みまで戻す。
			t.Cleanup(func() { _ = os.Chmod(actions, 0o700) }) //nolint:gosec // 削除に書き込み権限が要る

			_, err := ghfiles.Collect(root)
			assert.ErrorIs(t, err, os.ErrPermission)
		})

		t.Run("workflows の外に置かれた YAML は集めない", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeFile(t, filepath.Join(root, ".github", "ci.yaml"))
			writeFile(t, filepath.Join(root, ".github", "workflows", "nested", "ci.yaml"))

			got, err := ghfiles.Collect(root)
			require.NoError(t, err)
			assert.Empty(t, got)
		})
	})
}
