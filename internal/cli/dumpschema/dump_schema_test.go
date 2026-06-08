package dumpschema

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	mock_dumpschema "go-boilerplate/internal/cli/dumpschema/mock"
	"go-boilerplate/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const testWorkDir = "/work"

func newTestGenerator(t *testing.T, fs FileSystem, runner CommandRunner) *generator {
	t.Helper()
	return &generator{
		logger:          logging.NewTestLogger(t),
		callerSkipCount: 1,
		permission:      schemaFilePerm,
		workDir:         testWorkDir,
		schemaRelPath:   "database/gen/schema.gen.sql",
		dumpCommand:     dumpCommand,
		dumpArgs:        dumpSubArgs,
		fs:              fs,
		runner:          runner,
	}
}

func TestGenerator_dumpSchema(t *testing.T) {
	t.Parallel()

	schemaAbs := filepath.Join(testWorkDir, "database/gen/schema.gen.sql")

	t.Run("正常系_pg_dumpの出力がschemaファイルへ書き込まれる", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_dumpschema.NewMockFileSystem(ctrl)
		runner := mock_dumpschema.NewMockCommandRunner(ctrl)

		out := []byte("CREATE TABLE users (id int);\n")
		runner.EXPECT().Run(gomock.Any(), testWorkDir, "pg_dump", gomock.Any()).Return(out, nil)
		fs.EXPECT().WriteFile(schemaAbs, out, os.FileMode(schemaFilePerm)).Return(nil)

		g := newTestGenerator(t, fs, runner)
		require.NoError(t, g.dumpSchema(context.Background(), "postgres://dsn"))
	})

	t.Run("異常系_pg_dump失敗時はWriteFileを呼ばずエラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_dumpschema.NewMockFileSystem(ctrl)
		runner := mock_dumpschema.NewMockCommandRunner(ctrl)

		runner.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("pg_dump failed"))
		// fs.WriteFile は EXPECT しない（呼ばれたら gomock が失敗させる）。

		g := newTestGenerator(t, fs, runner)
		require.Error(t, g.dumpSchema(context.Background(), "postgres://dsn"))
	})

	t.Run("異常系_書き込み失敗時はエラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_dumpschema.NewMockFileSystem(ctrl)
		runner := mock_dumpschema.NewMockCommandRunner(ctrl)

		runner.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("x"), nil)
		fs.EXPECT().WriteFile(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("write failed"))

		g := newTestGenerator(t, fs, runner)
		require.Error(t, g.dumpSchema(context.Background(), "postgres://dsn"))
	})
}

func TestGenerator_sanitizeSchemaInPlace(t *testing.T) {
	t.Parallel()

	schemaAbs := filepath.Join(testWorkDir, "database/gen/schema.gen.sql")

	t.Run("正常系_メタコマンドとバージョンコメントと空行を除去しDDLは残す", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_dumpschema.NewMockFileSystem(ctrl)

		input := "" +
			"\\connect mydb\n" +
			"-- Dumped from database version 14.1\n" +
			"-- Dumped by pg_dump version 14.1\n" +
			"\n" +
			"CREATE TABLE users (id int);\n" +
			"\n" +
			"CREATE TABLE items (id int);\n"
		fs.EXPECT().ReadFile(schemaAbs).Return([]byte(input), nil)

		var written []byte
		fs.EXPECT().WriteFile(schemaAbs, gomock.Any(), os.FileMode(schemaFilePerm)).DoAndReturn(
			func(_ string, data []byte, _ os.FileMode) error {
				written = data
				return nil
			})

		g := newTestGenerator(t, fs, mock_dumpschema.NewMockCommandRunner(ctrl))
		require.NoError(t, g.sanitizeSchemaInPlace())

		got := string(written)
		assert.Contains(t, got, "CREATE TABLE users (id int);")
		assert.Contains(t, got, "CREATE TABLE items (id int);")
		assert.NotContains(t, got, "\\connect")
		assert.NotContains(t, got, "Dumped from database version")
		assert.NotContains(t, got, "Dumped by pg_dump version")
	})

	t.Run("異常系_読み込み失敗時はエラーで書き込まない", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_dumpschema.NewMockFileSystem(ctrl)

		fs.EXPECT().ReadFile(gomock.Any()).Return(nil, errors.New("read failed"))
		// WriteFile は呼ばれない。

		g := newTestGenerator(t, fs, mock_dumpschema.NewMockCommandRunner(ctrl))
		require.Error(t, g.sanitizeSchemaInPlace())
	})

	t.Run("異常系_書き込み失敗時はエラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_dumpschema.NewMockFileSystem(ctrl)

		fs.EXPECT().ReadFile(gomock.Any()).Return([]byte("CREATE TABLE x (id int);\n"), nil)
		fs.EXPECT().WriteFile(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("write failed"))

		g := newTestGenerator(t, fs, mock_dumpschema.NewMockCommandRunner(ctrl))
		require.Error(t, g.sanitizeSchemaInPlace())
	})
}

func TestNewGenerator(t *testing.T) {
	t.Parallel()

	g := newGenerator(logging.NewTestLogger(t), "/app")
	assert.Equal(t, "/app", g.workDir)
	assert.Equal(t, "database/gen/schema.gen.sql", g.schemaRelPath)
	assert.Equal(t, "pg_dump", g.dumpCommand)
	assert.Equal(t, dumpSubArgs, g.dumpArgs)
	assert.Equal(t, os.FileMode(schemaFilePerm), g.permission)
	assert.NotNil(t, g.fs)
	assert.NotNil(t, g.runner)
}
