package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_run(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("必須フラグが揃えば出力先に生成する", func(t *testing.T) {
			t.Parallel()
			outDir := t.TempDir()

			require.NoError(t, run([]string{"-name", "request id", "-type", "string", "-out", outDir}))
			assert.FileExists(t, filepath.Join(outDir, "requestid_ctx.gen.go"))
			assert.FileExists(t, filepath.Join(outDir, "requestid_ctx_test.go"))
		})

		t.Run("import と alias と test-value を生成器へ渡す", func(t *testing.T) {
			t.Parallel()
			outDir := t.TempDir()

			require.NoError(t, run([]string{
				"-name", "trace id",
				"-type", "uuid.UUID",
				"-import", "github.com/google/uuid",
				"-alias", "uuid",
				"-test-value", "uuid.New()",
				"-out", outDir,
			}))

			//nolint:gosec // path は t.TempDir と本ファイル内のリテラル
			body, err := os.ReadFile(filepath.Join(outDir, "traceid_ctx.gen.go"))
			require.NoError(t, err)
			assert.Contains(t, string(body), `uuid "github.com/google/uuid"`)
		})

		t.Run("ヘルプ要求は失敗にしない", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, run([]string{"-h"}))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("name が無ければ足りないフラグを示して失敗する", func(t *testing.T) {
			t.Parallel()
			outDir := t.TempDir()

			require.ErrorIs(t, run([]string{"-type", "string", "-out", outDir}), errNameRequired)
			assert.NoFileExists(t, filepath.Join(outDir, "requestid_ctx.gen.go"))
		})

		t.Run("type が無ければ足りないフラグを示して失敗する", func(t *testing.T) {
			t.Parallel()
			outDir := t.TempDir()

			require.ErrorIs(t, run([]string{"-name", "request id", "-out", outDir}), errTypeRequired)
			assert.NoFileExists(t, filepath.Join(outDir, "requestid_ctx.gen.go"))
		})

		t.Run("生成器の失敗をそのまま伝える", func(t *testing.T) {
			t.Parallel()

			err := run([]string{"-name", "1st", "-type", "string", "-out", t.TempDir()})

			require.ErrorIs(t, err, errInvalidIdentifier)
		})

		t.Run("未知のフラグはヘルプ要求と混同せず失敗する", func(t *testing.T) {
			t.Parallel()

			require.ErrorContains(t, run([]string{"-bogus"}), "failed to parse flags")
		})
	})
}
