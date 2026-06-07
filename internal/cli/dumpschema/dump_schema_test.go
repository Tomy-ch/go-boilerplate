package dumpschema

import (
	"os"
	"path/filepath"
	"testing"

	"go-boilerplate/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestGenerator(t *testing.T, workDir string) *generator {
	t.Helper()
	return &generator{
		logger:          logging.NewTestLogger(t),
		callerSkipCount: 1,
		permission:      schemaFilePerm,
		workDir:         workDir,
		schemaRelPath:   "database/gen/schema.gen.sql",
		dumpCommand:     dumpCommand,
		dumpArgs:        dumpSubArgs,
	}
}

func TestGenerator_sanitizeSchemaInPlace(t *testing.T) {
	t.Parallel()

	work := t.TempDir()
	schemaPath := filepath.Join(work, "database/gen/schema.gen.sql")
	require.NoError(t, os.MkdirAll(filepath.Dir(schemaPath), 0o750))

	// psql メタコマンド行・pg_dump バージョンコメント・空行が混在したスキーマを用意する。
	input := "" +
		"\\connect mydb\n" +
		"-- Dumped from database version 14.1\n" +
		"-- Dumped by pg_dump version 14.1\n" +
		"\n" +
		"CREATE TABLE users (id int);\n" +
		"\n" +
		"CREATE TABLE items (id int);\n"
	require.NoError(t, os.WriteFile(schemaPath, []byte(input), 0o600))

	g := newTestGenerator(t, work)
	require.NoError(t, g.sanitizeSchemaInPlace())

	out, err := os.ReadFile(schemaPath) //nolint:gosec // G304: テスト用の一時ディレクトリ配下のパスで外部入力ではない
	require.NoError(t, err)
	got := string(out)

	// 実 DDL は保持される。
	assert.Contains(t, got, "CREATE TABLE users (id int);")
	assert.Contains(t, got, "CREATE TABLE items (id int);")
	// メタコマンド行・バージョンコメントは除去される。
	assert.NotContains(t, got, "\\connect")
	assert.NotContains(t, got, "Dumped from database version")
	assert.NotContains(t, got, "Dumped by pg_dump version")
}

func TestGenerator_sanitizeSchemaInPlace_ReadError(t *testing.T) {
	t.Parallel()

	t.Run("異常系_schemaファイルが存在しない場合は読み込みエラー", func(t *testing.T) {
		t.Parallel()
		// schema ファイルが存在しない workDir を渡すと読み込みエラーになる。
		g := newTestGenerator(t, t.TempDir())
		err := g.sanitizeSchemaInPlace()
		require.Error(t, err)
	})
}

func TestNewGenerator(t *testing.T) {
	t.Parallel()

	t.Cleanup(func() { workDir = "" })
	workDir = "/app"
	g := newGenerator(logging.NewTestLogger(t))
	assert.Equal(t, "/app", g.workDir)
	assert.Equal(t, "database/gen/schema.gen.sql", g.schemaRelPath)
	assert.Equal(t, "pg_dump", g.dumpCommand)
	assert.Equal(t, dumpSubArgs, g.dumpArgs)
	assert.Equal(t, os.FileMode(schemaFilePerm), g.permission)
}
