package dumpschema

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go-boilerplate/internal/logging"
	mock_exec "go-boilerplate/pkg/exec/mock"
	mock_fs "go-boilerplate/pkg/fs/mock"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const testWorkDir = "/work"

func newTestGenerator(t *testing.T, fs *mock_fs.MockFS, runner *mock_exec.MockRunner) *Generator {
	t.Helper()
	return &Generator{
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

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("pg_dumpの出力を整形してschemaファイルへ書き込む", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fs := mock_fs.NewMockFS(ctrl)
			runner := mock_exec.NewMockRunner(ctrl)

			// メタコマンド行と末尾空行を含む生ダンプが、整形後のバイト列で 1 度だけ書き込まれることを検証する。
			out := []byte("\\connect mydb\nCREATE TABLE users (id int);\n")
			wantWritten := []byte("CREATE TABLE users (id int);")
			// DSN を先頭に dumpSubArgs を連結した引数と、PGPASSWORD を含む env が
			// そのまま runner へ渡ることを具体値で検証する。
			wantEnv := []string{"PGPASSWORD=pw"}
			wantArgs := []string{"postgres://dsn", "--schema-only", "--no-owner", "--no-privileges", "--format=plain"}
			runner.EXPECT().Output(gomock.Any(), testWorkDir, wantEnv, "pg_dump", wantArgs).Return(out, nil)
			fs.EXPECT().WriteFile(schemaAbs, wantWritten, os.FileMode(schemaFilePerm)).Return(nil)

			g := newTestGenerator(t, fs, runner)
			require.NoError(t, g.dumpSchema(context.Background(), "postgres://dsn", "pw"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("pg_dump失敗時はWriteFileを呼ばずエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fs := mock_fs.NewMockFS(ctrl)
			runner := mock_exec.NewMockRunner(ctrl)

			runner.EXPECT().Output(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, xerrors.New("pg_dump failed"))

			g := newTestGenerator(t, fs, runner)
			require.Error(t, g.dumpSchema(context.Background(), "postgres://dsn", "pw"))
		})

		t.Run("書き込み失敗時はエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fs := mock_fs.NewMockFS(ctrl)
			runner := mock_exec.NewMockRunner(ctrl)

			runner.EXPECT().Output(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("x"), nil)
			fs.EXPECT().WriteFile(gomock.Any(), gomock.Any(), gomock.Any()).Return(xerrors.New("write failed"))

			g := newTestGenerator(t, fs, runner)
			require.Error(t, g.dumpSchema(context.Background(), "postgres://dsn", "pw"))
		})
	})
}

func Test_sanitizeSchema(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("メタコマンドとバージョンコメントと空行を除去しDDLは残す", func(t *testing.T) {
			t.Parallel()

			input := []byte("" +
				"\\connect mydb\n" +
				"-- Dumped from database version 14.1\n" +
				"-- Dumped by pg_dump version 14.1\n" +
				"\n" +
				"CREATE TABLE users (id int);\n" +
				"\n" +
				"CREATE TABLE items (id int);\n")

			got := string(sanitizeSchema(input))
			assert.Contains(t, got, "CREATE TABLE users (id int);")
			assert.Contains(t, got, "CREATE TABLE items (id int);")
			assert.NotContains(t, got, "\\connect")
			assert.NotContains(t, got, "Dumped from database version")
			assert.NotContains(t, got, "Dumped by pg_dump version")
			assert.NotContains(t, got, "\n\n")
		})

		t.Run("整形対象が無い入力はDDL行がそのまま残る", func(t *testing.T) {
			t.Parallel()

			got := sanitizeSchema([]byte("CREATE TABLE x (id int);\nCREATE TABLE y (id int);\n"))
			assert.Equal(t, "CREATE TABLE x (id int);\nCREATE TABLE y (id int);", string(got))
		})
	})
}

func TestNewGenerator(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("workDirと固定設定とFS/Runner注入が宣言通りに行われる", func(t *testing.T) {
			t.Parallel()

			g := NewGenerator(logging.NewTestLogger(t), "/app")
			assert.Equal(t, "/app", g.workDir)
			assert.Equal(t, "database/gen/schema.gen.sql", g.schemaRelPath)
			assert.Equal(t, "pg_dump", g.dumpCommand)
			assert.Equal(t, dumpSubArgs, g.dumpArgs)
			assert.Equal(t, os.FileMode(schemaFilePerm), g.permission)
			assert.NotNil(t, g.fs)
			assert.NotNil(t, g.runner)
		})
	})
}

func TestGenerator_RunDump(t *testing.T) {
	t.Parallel()

	schemaAbs := filepath.Join(testWorkDir, "database/gen/schema.gen.sql")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DSN解決後にダンプと整形を行い1度だけ書き込む", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fs := mock_fs.NewMockFS(ctrl)
			runner := mock_exec.NewMockRunner(ctrl)

			dumped := []byte("CREATE TABLE users (id int);\n")
			// 生成後に読み直さず、整形結果を 1 度だけ書き込むこと（ReadFile を呼ばないこと）を検証する。
			runner.EXPECT().Output(gomock.Any(), testWorkDir, gomock.Any(), "pg_dump", gomock.Any()).Return(dumped, nil)
			fs.EXPECT().WriteFile(schemaAbs, []byte("CREATE TABLE users (id int);"), os.FileMode(schemaFilePerm)).Return(nil)

			g := newTestGenerator(t, fs, runner)
			loadDSN := func() (string, string, error) { return "postgres://dsn", "pw", nil }
			require.NoError(t, g.RunDump(context.Background(), loadDSN))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DSN解決に失敗するとダンプせずエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fs := mock_fs.NewMockFS(ctrl)
			runner := mock_exec.NewMockRunner(ctrl)

			g := newTestGenerator(t, fs, runner)
			loadDSN := func() (string, string, error) { return "", "", xerrors.New("config failed") }
			require.Error(t, g.RunDump(context.Background(), loadDSN))
		})

		t.Run("ダンプに失敗すると書き込まずエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fs := mock_fs.NewMockFS(ctrl)
			runner := mock_exec.NewMockRunner(ctrl)

			runner.EXPECT().Output(gomock.Any(), testWorkDir, gomock.Any(), "pg_dump", gomock.Any()).Return(nil, xerrors.New("pg_dump failed"))

			g := newTestGenerator(t, fs, runner)
			loadDSN := func() (string, string, error) { return "postgres://dsn", "pw", nil }
			require.Error(t, g.RunDump(context.Background(), loadDSN))
		})

		t.Run("書き込みに失敗するとエラーを伝播する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fs := mock_fs.NewMockFS(ctrl)
			runner := mock_exec.NewMockRunner(ctrl)

			dumped := []byte("CREATE TABLE users (id int);\n")
			runner.EXPECT().Output(gomock.Any(), testWorkDir, gomock.Any(), "pg_dump", gomock.Any()).Return(dumped, nil)
			fs.EXPECT().WriteFile(schemaAbs, gomock.Any(), os.FileMode(schemaFilePerm)).Return(xerrors.New("write failed"))

			g := newTestGenerator(t, fs, runner)
			loadDSN := func() (string, string, error) { return "postgres://dsn", "pw", nil }
			require.Error(t, g.RunDump(context.Background(), loadDSN))
		})
	})
}
