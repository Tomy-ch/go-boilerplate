package mergedml

import (
	"context"
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

func newTestGenerator(t *testing.T, fs FileSystem) *Generator {
	t.Helper()
	return &Generator{
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

	t.Run("異常系_連結中のReadFileに失敗するとエラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_mergedml.NewMockFileSystem(ctrl)

		f1 := filepath.Join(dmlDir, "001.sql")
		fs.EXPECT().FindSQLFiles(dmlDir).Return([]string{f1}, nil)
		fs.EXPECT().ReadFile(f1).Return(nil, errors.New("read failed"))

		g := newTestGenerator(t, fs)
		require.Error(t, g.buildCategorySQLFile("user", "repository"))
	})

	t.Run("異常系_連結結果の書き出しに失敗するとエラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_mergedml.NewMockFileSystem(ctrl)

		f1 := filepath.Join(dmlDir, "001.sql")
		fs.EXPECT().FindSQLFiles(dmlDir).Return([]string{f1}, nil)
		fs.EXPECT().ReadFile(f1).Return([]byte("SELECT 1;"), nil)
		fs.EXPECT().WriteFile(dstPath, gomock.Any(), os.FileMode(genFilePerm)).Return(errors.New("write failed"))

		g := newTestGenerator(t, fs)
		require.Error(t, g.buildCategorySQLFile("user", "repository"))
	})

	t.Run("正常系_SQLが空で生成物が未存在ならNotExistを無視する", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_mergedml.NewMockFileSystem(ctrl)

		fs.EXPECT().FindSQLFiles(dmlDir).Return(nil, nil)
		fs.EXPECT().Remove(dstPath).Return(os.ErrNotExist)

		g := newTestGenerator(t, fs)
		require.NoError(t, g.buildCategorySQLFile("user", "repository"))
	})

	t.Run("異常系_SQLが空でRemoveがNotExist以外で失敗するとエラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_mergedml.NewMockFileSystem(ctrl)

		fs.EXPECT().FindSQLFiles(dmlDir).Return(nil, nil)
		fs.EXPECT().Remove(dstPath).Return(errors.New("remove failed"))

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

	t.Run("異常系_stale削除に失敗するとエラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_mergedml.NewMockFileSystem(ctrl)

		fs.EXPECT().ListGenFileNames(genAbs).Return([]string{"old_repository.gen.sql"}, nil)
		fs.EXPECT().Remove(filepath.Join(genAbs, "old_repository.gen.sql")).Return(errors.New("remove failed"))

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

func TestResolveConcurrency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		numCPU int
		want   int
	}{
		// CPU 数が下限未満でも下限 minSQLCConcurrency を保証する（実行時値へクランプ）。
		{name: "正常系_CPU数が下限未満なら下限まで引き上げる", numCPU: 1, want: minSQLCConcurrency},
		// CPU 数が下限ちょうどならそのまま。
		{name: "正常系_CPU数が下限と同じならそのまま", numCPU: minSQLCConcurrency, want: minSQLCConcurrency},
		// 下限と上限の間ならその CPU 数を採用する。
		{name: "正常系_CPU数が下限と上限の間ならそのCPU数を採用する", numCPU: maxSQLCConcurrency - 1, want: maxSQLCConcurrency - 1},
		// CPU 数が上限ちょうどなら上限。
		{name: "正常系_CPU数が上限と同じなら上限を採用する", numCPU: maxSQLCConcurrency, want: maxSQLCConcurrency},
		// CPU 数が上限超過なら上限で頭打ちにする。
		{name: "正常系_CPU数が上限超過でも上限で頭打ちにする", numCPU: maxSQLCConcurrency + 8, want: maxSQLCConcurrency},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, resolveConcurrency(tt.numCPU))
		})
	}
}

func TestNewGenerator(t *testing.T) {
	t.Parallel()

	g := NewGenerator(logging.NewTestLogger(t), "/custom")
	assert.Equal(t, "/custom", g.workDir)
	assert.Equal(t, "database/dml/", g.dmlRootDir)
	assert.Equal(t, "database/gen/", g.genRootDir)
	assert.Equal(t, 1, g.callerSkipCount)
	assert.NotNil(t, g.fs)
}

func TestRunMerge(t *testing.T) {
	t.Parallel()

	const targetType = "repository"
	typeRoot := filepath.Join(testWorkDir, "database/dml/", targetType)
	genAbs := filepath.Join(testWorkDir, "database/gen/")

	t.Run("正常系_カテゴリを並列マージしstaleを掃除する", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_mergedml.NewMockFileSystem(ctrl)

		userDir := filepath.Join(testWorkDir, "database/dml/", targetType, "user")
		userSQL := filepath.Join(userDir, "001.sql")
		dst := filepath.Join(genAbs, "user_repository.gen.sql")

		fs.EXPECT().ListSubDirNames(typeRoot).Return([]string{"user"}, nil)
		fs.EXPECT().FindSQLFiles(userDir).Return([]string{userSQL}, nil)
		fs.EXPECT().ReadFile(userSQL).Return([]byte("SELECT 1;"), nil)
		fs.EXPECT().WriteFile(dst, gomock.Any(), os.FileMode(genFilePerm)).Return(nil)
		fs.EXPECT().ListGenFileNames(genAbs).Return([]string{"user_repository.gen.sql"}, nil)

		g := newTestGenerator(t, fs)
		require.NoError(t, RunMerge(context.Background(), g, targetType))
	})

	t.Run("正常系_カテゴリ0件のときはcleanupのみ実行する", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_mergedml.NewMockFileSystem(ctrl)

		fs.EXPECT().ListSubDirNames(typeRoot).Return(nil, nil)
		// 0件でも同 type の stale は全消し対象になる。
		fs.EXPECT().ListGenFileNames(genAbs).Return([]string{"old_repository.gen.sql"}, nil)
		fs.EXPECT().Remove(filepath.Join(genAbs, "old_repository.gen.sql")).Return(nil)

		g := newTestGenerator(t, fs)
		require.NoError(t, RunMerge(context.Background(), g, targetType))
	})

	t.Run("異常系_カテゴリ一覧の取得に失敗するとエラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_mergedml.NewMockFileSystem(ctrl)

		fs.EXPECT().ListSubDirNames(typeRoot).Return(nil, errors.New("read dir failed"))

		g := newTestGenerator(t, fs)
		require.Error(t, RunMerge(context.Background(), g, targetType))
	})

	t.Run("異常系_カテゴリ0件かつcleanupに失敗するとエラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_mergedml.NewMockFileSystem(ctrl)

		fs.EXPECT().ListSubDirNames(typeRoot).Return(nil, nil)
		fs.EXPECT().ListGenFileNames(genAbs).Return(nil, errors.New("list failed"))

		g := newTestGenerator(t, fs)
		require.Error(t, RunMerge(context.Background(), g, targetType))
	})

	t.Run("異常系_カテゴリのマージに失敗するとエラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_mergedml.NewMockFileSystem(ctrl)

		userDir := filepath.Join(testWorkDir, "database/dml/", targetType, "user")
		fs.EXPECT().ListSubDirNames(typeRoot).Return([]string{"user"}, nil)
		fs.EXPECT().FindSQLFiles(userDir).Return(nil, errors.New("walk failed"))

		g := newTestGenerator(t, fs)
		require.Error(t, RunMerge(context.Background(), g, targetType))
	})

	t.Run("異常系_ctxキャンセル済みならセマフォ取得に失敗しエラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_mergedml.NewMockFileSystem(ctrl)

		// カテゴリは存在するが、ctx が先にキャンセルされているため sem.Acquire が失敗し、
		// 各カテゴリの走査(FindSQLFiles)へは進まない。
		fs.EXPECT().ListSubDirNames(typeRoot).Return([]string{"user"}, nil)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		g := newTestGenerator(t, fs)
		require.Error(t, RunMerge(ctx, g, targetType))
	})

	t.Run("異常系_マージ成功後のcleanupに失敗するとエラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fs := mock_mergedml.NewMockFileSystem(ctrl)

		userDir := filepath.Join(testWorkDir, "database/dml/", targetType, "user")
		userSQL := filepath.Join(userDir, "001.sql")
		dst := filepath.Join(genAbs, "user_repository.gen.sql")

		fs.EXPECT().ListSubDirNames(typeRoot).Return([]string{"user"}, nil)
		fs.EXPECT().FindSQLFiles(userDir).Return([]string{userSQL}, nil)
		fs.EXPECT().ReadFile(userSQL).Return([]byte("SELECT 1;"), nil)
		fs.EXPECT().WriteFile(dst, gomock.Any(), os.FileMode(genFilePerm)).Return(nil)
		fs.EXPECT().ListGenFileNames(genAbs).Return(nil, errors.New("list failed"))

		g := newTestGenerator(t, fs)
		require.Error(t, RunMerge(context.Background(), g, targetType))
	})
}

func TestOSFileSystem(t *testing.T) {
	t.Parallel()

	var sut osFileSystem

	t.Run("ListSubDirNames_サブディレクトリ名を昇順で返しファイルは除外する", func(t *testing.T) {
		t.Parallel()
		base := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(base, "b"), 0o750))
		require.NoError(t, os.Mkdir(filepath.Join(base, "a"), 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(base, "file.txt"), []byte("x"), 0o600))

		dirs, err := sut.ListSubDirNames(base)
		require.NoError(t, err)
		assert.Equal(t, []string{"a", "b"}, dirs)
	})

	t.Run("ListSubDirNames_存在しないディレクトリはエラー", func(t *testing.T) {
		t.Parallel()
		_, err := sut.ListSubDirNames(filepath.Join(t.TempDir(), "missing"))
		require.Error(t, err)
	})

	t.Run("ListGenFileNames_ファイル名のみ返しディレクトリは除外する", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.sql"), []byte("x"), 0o600))
		require.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0o750))

		names, err := sut.ListGenFileNames(dir)
		require.NoError(t, err)
		assert.Equal(t, []string{"a.sql"}, names)
	})

	t.Run("ListGenFileNames_存在しないディレクトリはエラー", func(t *testing.T) {
		t.Parallel()
		_, err := sut.ListGenFileNames(filepath.Join(t.TempDir(), "missing"))
		require.Error(t, err)
	})

	t.Run("FindSQLFiles_配下のsqlのみ昇順で返す", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		sub := filepath.Join(root, "sub")
		require.NoError(t, os.Mkdir(sub, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(root, "b.sql"), []byte("x"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(sub, "a.sql"), []byte("x"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(root, "note.txt"), []byte("x"), 0o600))

		files, err := sut.FindSQLFiles(root)
		require.NoError(t, err)
		// フルパス文字列の昇順（"…/b.sql" < "…/sub/a.sql"）で返ること。
		assert.Equal(t, []string{filepath.Join(root, "b.sql"), filepath.Join(sub, "a.sql")}, files)
	})

	t.Run("FindSQLFiles_存在しないルートはエラー", func(t *testing.T) {
		t.Parallel()
		_, err := sut.FindSQLFiles(filepath.Join(t.TempDir(), "missing"))
		require.Error(t, err)
	})

	t.Run("ReadFile_WriteFile_往復で同じ内容を読み書きできる", func(t *testing.T) {
		t.Parallel()
		p := filepath.Join(t.TempDir(), "x.sql")
		require.NoError(t, sut.WriteFile(p, []byte("SELECT 1;"), 0o600))

		b, err := sut.ReadFile(p)
		require.NoError(t, err)
		assert.Equal(t, "SELECT 1;", string(b))
	})

	t.Run("ReadFile_存在しないファイルはエラー", func(t *testing.T) {
		t.Parallel()
		_, err := sut.ReadFile(filepath.Join(t.TempDir(), "missing.sql"))
		require.Error(t, err)
	})

	t.Run("Remove_ファイルを削除できる", func(t *testing.T) {
		t.Parallel()
		p := filepath.Join(t.TempDir(), "x.sql")
		require.NoError(t, os.WriteFile(p, []byte("x"), 0o600))
		require.NoError(t, sut.Remove(p))

		_, err := os.Stat(p)
		require.Error(t, err)
	})

	t.Run("Remove_存在しないファイルはエラー", func(t *testing.T) {
		t.Parallel()
		require.Error(t, sut.Remove(filepath.Join(t.TempDir(), "missing.sql")))
	})
}
