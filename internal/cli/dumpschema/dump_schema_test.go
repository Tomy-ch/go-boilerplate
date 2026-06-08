package dumpschema

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"go-boilerplate/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nopWriteCloser は io.Writer を io.WriteCloser に変換するテスト用ラッパです。
type nopWriteCloser struct {
	io.Writer
	closeErr error
}

// fakeFS は fileSystem のフェイク実装。実ファイルシステムに一切触れません。
type fakeFS struct {
	createBuf *bytes.Buffer // Create が返す書き込み先（pg_dump 出力の捕捉用）
	createErr error
	closeErr  error

	readData []byte
	readErr  error

	written     []byte // WriteFile に渡された内容
	writtenPerm os.FileMode
	writeErr    error
	writeCalled bool
}

// fakeRunner は commandRunner のフェイク実装。pg_dump を実行しません。
type fakeRunner struct {
	output []byte // stdout へ書き込む内容
	err    error
	called bool
}

func (n nopWriteCloser) Close() error { return n.closeErr }

func (f *fakeFS) Create(string) (io.WriteCloser, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.createBuf = &bytes.Buffer{}
	return nopWriteCloser{Writer: f.createBuf, closeErr: f.closeErr}, nil
}

func (f *fakeFS) ReadFile(string) ([]byte, error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	return f.readData, nil
}

func (f *fakeFS) WriteFile(_ string, data []byte, perm os.FileMode) error {
	f.writeCalled = true
	if f.writeErr != nil {
		return f.writeErr
	}
	f.written = data
	f.writtenPerm = perm
	return nil
}

func (r *fakeRunner) Run(_ context.Context, _, _ string, _ []string, stdout io.Writer) error {
	r.called = true
	if r.err != nil {
		return r.err
	}
	if r.output != nil {
		_, _ = stdout.Write(r.output)
	}
	return nil
}

func newTestGenerator(t *testing.T, fs fileSystem, runner commandRunner) *generator {
	t.Helper()
	return &generator{
		logger:          logging.NewTestLogger(t),
		callerSkipCount: 1,
		permission:      schemaFilePerm,
		workDir:         "/work",
		schemaRelPath:   "database/gen/schema.gen.sql",
		dumpCommand:     dumpCommand,
		dumpArgs:        dumpSubArgs,
		fs:              fs,
		runner:          runner,
	}
}

func TestGenerator_dumpSchema(t *testing.T) {
	t.Parallel()

	t.Run("正常系_pg_dumpの出力が作成ファイルへ書き込まれる", func(t *testing.T) {
		t.Parallel()
		fs := &fakeFS{}
		runner := &fakeRunner{output: []byte("CREATE TABLE users (id int);\n")}

		g := newTestGenerator(t, fs, runner)
		require.NoError(t, g.dumpSchema(context.Background(), "postgres://dsn"))

		assert.True(t, runner.called, "runner が呼ばれること")
		require.NotNil(t, fs.createBuf)
		assert.Contains(t, fs.createBuf.String(), "CREATE TABLE users (id int);")
	})

	t.Run("異常系_出力ファイル作成に失敗するとエラー", func(t *testing.T) {
		t.Parallel()
		fs := &fakeFS{createErr: errors.New("create failed")}
		runner := &fakeRunner{}

		g := newTestGenerator(t, fs, runner)
		err := g.dumpSchema(context.Background(), "postgres://dsn")

		require.Error(t, err)
		assert.False(t, runner.called, "作成失敗時は runner を呼ばないこと")
	})

	t.Run("異常系_pg_dump実行に失敗するとエラー", func(t *testing.T) {
		t.Parallel()
		fs := &fakeFS{}
		runner := &fakeRunner{err: errors.New("pg_dump failed")}

		g := newTestGenerator(t, fs, runner)
		err := g.dumpSchema(context.Background(), "postgres://dsn")

		require.Error(t, err)
	})
}

func TestGenerator_sanitizeSchemaInPlace(t *testing.T) {
	t.Parallel()

	t.Run("正常系_メタコマンドとバージョンコメントと空行を除去しDDLは残す", func(t *testing.T) {
		t.Parallel()
		input := "" +
			"\\connect mydb\n" +
			"-- Dumped from database version 14.1\n" +
			"-- Dumped by pg_dump version 14.1\n" +
			"\n" +
			"CREATE TABLE users (id int);\n" +
			"\n" +
			"CREATE TABLE items (id int);\n"
		fs := &fakeFS{readData: []byte(input)}

		g := newTestGenerator(t, fs, &fakeRunner{})
		require.NoError(t, g.sanitizeSchemaInPlace())

		require.True(t, fs.writeCalled, "整形結果が書き戻されること")
		got := string(fs.written)
		assert.Contains(t, got, "CREATE TABLE users (id int);")
		assert.Contains(t, got, "CREATE TABLE items (id int);")
		assert.NotContains(t, got, "\\connect")
		assert.NotContains(t, got, "Dumped from database version")
		assert.NotContains(t, got, "Dumped by pg_dump version")
		assert.Equal(t, os.FileMode(schemaFilePerm), fs.writtenPerm)
	})

	t.Run("異常系_読み込み失敗時はエラーで書き込まない", func(t *testing.T) {
		t.Parallel()
		fs := &fakeFS{readErr: errors.New("read failed")}

		g := newTestGenerator(t, fs, &fakeRunner{})
		err := g.sanitizeSchemaInPlace()

		require.Error(t, err)
		assert.False(t, fs.writeCalled, "読み込み失敗時は書き込まないこと")
	})

	t.Run("異常系_書き込み失敗時はエラー", func(t *testing.T) {
		t.Parallel()
		fs := &fakeFS{readData: []byte("CREATE TABLE x (id int);\n"), writeErr: errors.New("write failed")}

		g := newTestGenerator(t, fs, &fakeRunner{})
		err := g.sanitizeSchemaInPlace()

		require.Error(t, err)
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
