package mergedml

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mock_mergedml "go-boilerplate/internal/cli/mergedml/mock"
	"go-boilerplate/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const testWorkDir = "/work"

func newTestGenerator(t *testing.T, fs FileSystem) *generator {
	t.Helper()
	return &generator{
		logger:          logging.NewTestLogger(t),
		callerSkipCount: 1,
		workDir:         testWorkDir,
		dmlRootDir:      "database/dml/",
		genRootDir:      "database/gen/",
		sqlcCfg:         "sqlc.yaml",
		fs:              fs,
	}
}

func TestGenerator_buildCategorySQLFile(t *testing.T) {
	t.Parallel()

	dmlDir := filepath.Join(testWorkDir, "database/dml/", "repository", "user")
	dstPath := filepath.Join(testWorkDir, "database/gen/", "user_repository.gen.sql")

	t.Run("正常系_カテゴリ内のsqlを昇順で連結して書き出す", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_mergedml.NewMockFileSystem(ctrl)

		f1 := filepath.Join(dmlDir, "001.sql")
		f2 := filepath.Join(dmlDir, "002.sql")
		fs.EXPECT().FindSQLFiles(dmlDir).Return([]string{f1, f2}, nil)
		fs.EXPECT().ReadFile(f1).Return([]byte("SELECT 1;"), nil)
		fs.EXPECT().ReadFile(f2).Return([]byte("SELECT 2;"), nil)

		var written []byte
		fs.EXPECT().WriteFile(dstPath, gomock.Any(), os.FileMode(genFilePerm)).DoAndReturn(
			func(_ string, data []byte, _ os.FileMode) error {
				written = data
				return nil
			})

		g := newTestGenerator(t, fs)
		require.NoError(t, g.buildCategorySQLFile("user", "repository"))

		got := string(written)
		assert.Contains(t, got, "SELECT 1;")
		assert.Contains(t, got, "SELECT 2;")
		assert.Less(t, strings.Index(got, "SELECT 1;"), strings.Index(got, "SELECT 2;"))
		assert.Contains(t, got, "-- === source:")
		// 末尾改行が無い入力でも連結が壊れないこと。
		assert.NotContains(t, got, "SELECT 1;SELECT 2;")
	})

	t.Run("正常系_SQLが空のカテゴリは生成物を削除する", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_mergedml.NewMockFileSystem(ctrl)

		fs.EXPECT().FindSQLFiles(dmlDir).Return(nil, nil)
		fs.EXPECT().Remove(dstPath).Return(nil)

		g := newTestGenerator(t, fs)
		require.NoError(t, g.buildCategorySQLFile("user", "repository"))
	})

	t.Run("異常系_入力走査に失敗するとエラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_mergedml.NewMockFileSystem(ctrl)

		fs.EXPECT().FindSQLFiles(dmlDir).Return(nil, errors.New("walk failed"))

		g := newTestGenerator(t, fs)
		require.Error(t, g.buildCategorySQLFile("user", "repository"))
	})
}

func TestGenerator_cleanupStaleGeneratedFiles(t *testing.T) {
	t.Parallel()

	genAbs := filepath.Join(testWorkDir, "database/gen/")

	t.Run("正常系_keep対象は残し同typeのstaleのみ削除し他typeは触らない", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_mergedml.NewMockFileSystem(ctrl)

		fs.EXPECT().ListGenFileNames(genAbs).Return([]string{
			"user_repository.gen.sql",   // keep（今回の生成対象）
			"old_repository.gen.sql",    // stale（同 type・keep外）→削除
			"foo_query_service.gen.sql", // 別 type →非対象で温存
		}, nil)
		// 削除されるのは old_repository のみ（他が Remove されたら gomock が失敗させる）。
		fs.EXPECT().Remove(filepath.Join(genAbs, "old_repository.gen.sql")).Return(nil)

		g := newTestGenerator(t, fs)
		require.NoError(t, g.cleanupStaleGeneratedFiles([]string{"user"}, "repository"))
	})

	t.Run("正常系_カテゴリ0件のとき同typeの生成物を全削除し他typeは温存する", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_mergedml.NewMockFileSystem(ctrl)

		fs.EXPECT().ListGenFileNames(genAbs).Return([]string{
			"a_repository.gen.sql",
			"b_repository.gen.sql",
			"x_query_service.gen.sql",
		}, nil)
		fs.EXPECT().Remove(filepath.Join(genAbs, "a_repository.gen.sql")).Return(nil)
		fs.EXPECT().Remove(filepath.Join(genAbs, "b_repository.gen.sql")).Return(nil)

		g := newTestGenerator(t, fs)
		require.NoError(t, g.cleanupStaleGeneratedFiles(nil, "repository"))
	})

	t.Run("異常系_一覧取得に失敗するとエラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_mergedml.NewMockFileSystem(ctrl)

		fs.EXPECT().ListGenFileNames(genAbs).Return(nil, errors.New("read failed"))

		g := newTestGenerator(t, fs)
		require.Error(t, g.cleanupStaleGeneratedFiles([]string{"user"}, "repository"))
	})
}

func TestGenerator_ensureUnderDir(t *testing.T) {
	t.Parallel()

	g := newTestGenerator(t, nil)

	t.Run("正常系_genRootDir配下のパスは許容される", func(t *testing.T) {
		t.Parallel()
		err := g.ensureUnderDir(filepath.Join(testWorkDir, "database/gen", "user_repository.gen.sql"))
		require.NoError(t, err)
	})

	t.Run("異常系_genRootDirの外を指すパスはエラー", func(t *testing.T) {
		t.Parallel()
		err := g.ensureUnderDir(filepath.Join(testWorkDir, "database", "outside.sql"))
		require.Error(t, err)
	})
}

func TestGenerator_dmlTypeRootAbs(t *testing.T) {
	t.Parallel()

	g := newTestGenerator(t, nil)
	assert.Equal(t, filepath.Join(testWorkDir, "database/dml/", "repository"), g.dmlTypeRootAbs("repository"))
}

func TestResolveConcurrencyConst(t *testing.T) {
	t.Parallel()

	got := resolveConcurrencyConst()
	assert.GreaterOrEqual(t, got, 1)
	assert.LessOrEqual(t, got, maxSQLCConcurrency)
}

func TestNewGenerator(t *testing.T) {
	t.Parallel()

	g := newGenerator(logging.NewTestLogger(t), "/custom")
	assert.Equal(t, "/custom", g.workDir)
	assert.Equal(t, "database/dml/", g.dmlRootDir)
	assert.Equal(t, "database/gen/", g.genRootDir)
	assert.Equal(t, 1, g.callerSkipCount)
	assert.NotNil(t, g.fs)
}
