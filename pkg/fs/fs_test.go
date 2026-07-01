package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOS_WriteFileAndReadFile(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("WriteFile後にReadFileで同じ内容が読める", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "sample.txt")
			sut := OS{}

			require.NoError(t, sut.WriteFile(path, []byte("hello"), 0o600))

			got, err := sut.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, "hello", string(got))
		})
	})
}

func TestOS_WriteFile(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("親ディレクトリが存在しないパスへの書き込みはエラーを返す", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "no-such-dir", "file.txt")
			err := OS{}.WriteFile(path, []byte("x"), 0o600)
			require.Error(t, err)
		})
	})
}

func TestOS_ReadFile_NotExist(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないファイルはエラーを返す", func(t *testing.T) {
			t.Parallel()
			_, err := OS{}.ReadFile(filepath.Join(t.TempDir(), "not-exist"))
			require.Error(t, err)
		})
	})
}

func TestOS_Glob(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("パターンにマッチするファイルのみを返す", func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "a.sql"), []byte("x"), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "b.sql"), []byte("x"), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "c.txt"), []byte("x"), 0o600))

			got, err := OS{}.Glob(filepath.Join(dir, "*.sql"))
			require.NoError(t, err)
			assert.Len(t, got, 2)
		})
	})
}
