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

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("カテゴリ内のsqlを昇順で連結して書き出す", func(t *testing.T) {
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

		t.Run("SQLが空のカテゴリは生成物を削除する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fs := mock_mergedml.NewMockFileSystem(ctrl)

			fs.EXPECT().FindSQLFiles(dmlDir).Return(nil, nil)
			fs.EXPECT().Remove(dstPath).Return(nil)

			g := newTestGenerator(t, fs)
			require.NoError(t, g.buildCategorySQLFile("user", "repository"))
		})

		t.Run("SQLが空で生成物が未存在ならNotExistを無視する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fs := mock_mergedml.NewMockFileSystem(ctrl)

			fs.EXPECT().FindSQLFiles(dmlDir).Return(nil, nil)
			fs.EXPECT().Remove(dstPath).Return(os.ErrNotExist)

			g := newTestGenerator(t, fs)
			require.NoError(t, g.buildCategorySQLFile("user", "repository"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("入力走査に失敗するとエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fs := mock_mergedml.NewMockFileSystem(ctrl)

			fs.EXPECT().FindSQLFiles(dmlDir).Return(nil, errors.New("walk failed"))

			g := newTestGenerator(t, fs)
			require.Error(t, g.buildCategorySQLFile("user", "repository"))
		})

		t.Run("連結中のReadFileに失敗するとエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fs := mock_mergedml.NewMockFileSystem(ctrl)

			f1 := filepath.Join(dmlDir, "001.sql")
			fs.EXPECT().FindSQLFiles(dmlDir).Return([]string{f1}, nil)
			fs.EXPECT().ReadFile(f1).Return(nil, errors.New("read failed"))

			g := newTestGenerator(t, fs)
			require.Error(t, g.buildCategorySQLFile("user", "repository"))
		})

		t.Run("連結結果の書き出しに失敗するとエラー", func(t *testing.T) {
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

		t.Run("SQLが空でRemoveがNotExist以外で失敗するとエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fs := mock_mergedml.NewMockFileSystem(ctrl)

			fs.EXPECT().FindSQLFiles(dmlDir).Return(nil, nil)
			fs.EXPECT().Remove(dstPath).Return(errors.New("remove failed"))

			g := newTestGenerator(t, fs)
			require.Error(t, g.buildCategorySQLFile("user", "repository"))
		})
	})
}

func TestGenerator_cleanupStaleGeneratedFiles(t *testing.T) {
	t.Parallel()

	genAbs := filepath.Join(testWorkDir, "database/gen/")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("keep対象は残し同typeのstaleのみ削除し他typeは触らない", func(t *testing.T) {
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

		t.Run("カテゴリ0件のとき同typeの生成物を全削除し他typeは温存する", func(t *testing.T) {
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
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("一覧取得に失敗するとエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fs := mock_mergedml.NewMockFileSystem(ctrl)

			fs.EXPECT().ListGenFileNames(genAbs).Return(nil, errors.New("read failed"))

			g := newTestGenerator(t, fs)
			require.Error(t, g.cleanupStaleGeneratedFiles([]string{"user"}, "repository"))
		})

		t.Run("stale削除に失敗するとエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fs := mock_mergedml.NewMockFileSystem(ctrl)

			fs.EXPECT().ListGenFileNames(genAbs).Return([]string{"old_repository.gen.sql"}, nil)
			fs.EXPECT().Remove(filepath.Join(genAbs, "old_repository.gen.sql")).Return(errors.New("remove failed"))

			g := newTestGenerator(t, fs)
			require.Error(t, g.cleanupStaleGeneratedFiles([]string{"user"}, "repository"))
		})
	})
}

func TestGenerator_ensureUnderDir(t *testing.T) {
	t.Parallel()

	g := newTestGenerator(t, nil)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("genRootDir配下のパスは許容される", func(t *testing.T) {
			t.Parallel()
			err := g.ensureUnderDir(filepath.Join(testWorkDir, "database/gen", "user_repository.gen.sql"))
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("genRootDirの外を指すパスはエラー", func(t *testing.T) {
			t.Parallel()
			err := g.ensureUnderDir(filepath.Join(testWorkDir, "database", "outside.sql"))
			require.Error(t, err)
		})
	})
}

func TestGenerator_dmlTypeRootAbs(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("workDirとdmlRootDirとtypeを結合した絶対パスを返す", func(t *testing.T) {
			t.Parallel()

			g := newTestGenerator(t, nil)
			assert.Equal(t, filepath.Join(testWorkDir, "database/dml/", "repository"), g.dmlTypeRootAbs("repository"))
		})
	})
}

func TestResolveConcurrencyConst(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("実行時CPU数を下限と上限の範囲内にクランプして返す", func(t *testing.T) {
			t.Parallel()

			got := resolveConcurrencyConst()
			assert.GreaterOrEqual(t, got, 1)
			assert.LessOrEqual(t, got, maxSQLCConcurrency)
		})
	})
}

func TestResolveConcurrency(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("CPU数が下限未満なら下限まで引き上げる", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, minSQLCConcurrency, resolveConcurrency(1))
		})
		t.Run("CPU数が下限と同じならそのまま", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, minSQLCConcurrency, resolveConcurrency(minSQLCConcurrency))
		})
		t.Run("CPU数が下限と上限の間ならそのCPU数を採用する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, maxSQLCConcurrency-1, resolveConcurrency(maxSQLCConcurrency-1))
		})
		t.Run("CPU数が上限と同じなら上限を採用する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, maxSQLCConcurrency, resolveConcurrency(maxSQLCConcurrency))
		})
		t.Run("CPU数が上限超過でも上限で頭打ちにする", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, maxSQLCConcurrency, resolveConcurrency(maxSQLCConcurrency+8))
		})
	})
}

func TestNewGenerator(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("workDirと固定ディレクトリ設定とFS注入が宣言通りに行われる", func(t *testing.T) {
			t.Parallel()

			g := NewGenerator(logging.NewTestLogger(t), "/custom")
			assert.Equal(t, "/custom", g.workDir)
			assert.Equal(t, "database/dml/", g.dmlRootDir)
			assert.Equal(t, "database/gen/", g.genRootDir)
			assert.Equal(t, 1, g.callerSkipCount)
			assert.NotNil(t, g.fs)
		})
	})
}

func TestRunMerge(t *testing.T) {
	t.Parallel()

	const targetType = "repository"
	typeRoot := filepath.Join(testWorkDir, "database/dml/", targetType)
	genAbs := filepath.Join(testWorkDir, "database/gen/")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("カテゴリを並列マージしstaleを掃除する", func(t *testing.T) {
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

		t.Run("カテゴリ0件のときはcleanupのみ実行する", func(t *testing.T) {
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
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("カテゴリ一覧の取得に失敗するとエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fs := mock_mergedml.NewMockFileSystem(ctrl)

			fs.EXPECT().ListSubDirNames(typeRoot).Return(nil, errors.New("read dir failed"))

			g := newTestGenerator(t, fs)
			require.Error(t, RunMerge(context.Background(), g, targetType))
		})

		t.Run("カテゴリ0件かつcleanupに失敗するとエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fs := mock_mergedml.NewMockFileSystem(ctrl)

			fs.EXPECT().ListSubDirNames(typeRoot).Return(nil, nil)
			fs.EXPECT().ListGenFileNames(genAbs).Return(nil, errors.New("list failed"))

			g := newTestGenerator(t, fs)
			require.Error(t, RunMerge(context.Background(), g, targetType))
		})

		t.Run("カテゴリのマージに失敗するとエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fs := mock_mergedml.NewMockFileSystem(ctrl)

			userDir := filepath.Join(testWorkDir, "database/dml/", targetType, "user")
			fs.EXPECT().ListSubDirNames(typeRoot).Return([]string{"user"}, nil)
			fs.EXPECT().FindSQLFiles(userDir).Return(nil, errors.New("walk failed"))

			g := newTestGenerator(t, fs)
			require.Error(t, RunMerge(context.Background(), g, targetType))
		})

		t.Run("ctxキャンセル済みならセマフォ取得に失敗しエラー", func(t *testing.T) {
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

		t.Run("マージ成功後のcleanupに失敗するとエラー", func(t *testing.T) {
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
	})
}

func TestOSFileSystem_ListSubDirNames(t *testing.T) {
	t.Parallel()

	var sut osFileSystem

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("サブディレクトリ名を昇順で返しファイルは除外する", func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(base, "b"), 0o750))
			require.NoError(t, os.Mkdir(filepath.Join(base, "a"), 0o750))
			require.NoError(t, os.WriteFile(filepath.Join(base, "file.txt"), []byte("x"), 0o600))

			dirs, err := sut.ListSubDirNames(base)
			require.NoError(t, err)
			assert.Equal(t, []string{"a", "b"}, dirs)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないディレクトリはエラー", func(t *testing.T) {
			t.Parallel()
			_, err := sut.ListSubDirNames(filepath.Join(t.TempDir(), "missing"))
			require.Error(t, err)
		})
	})
}

func TestOSFileSystem_ListGenFileNames(t *testing.T) {
	t.Parallel()

	var sut osFileSystem

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ファイル名のみ返しディレクトリは除外する", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "a.sql"), []byte("x"), 0o600))
			require.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0o750))

			names, err := sut.ListGenFileNames(dir)
			require.NoError(t, err)
			assert.Equal(t, []string{"a.sql"}, names)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないディレクトリはエラー", func(t *testing.T) {
			t.Parallel()
			_, err := sut.ListGenFileNames(filepath.Join(t.TempDir(), "missing"))
			require.Error(t, err)
		})
	})
}

func TestOSFileSystem_FindSQLFiles(t *testing.T) {
	t.Parallel()

	var sut osFileSystem

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("配下のsqlのみ昇順で返す", func(t *testing.T) {
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
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないルートはエラー", func(t *testing.T) {
			t.Parallel()
			_, err := sut.FindSQLFiles(filepath.Join(t.TempDir(), "missing"))
			require.Error(t, err)
		})
	})
}

func TestOSFileSystem_ReadFile(t *testing.T) {
	t.Parallel()

	var sut osFileSystem

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("WriteFileで書き込んだ内容を読み戻せる", func(t *testing.T) {
			t.Parallel()
			p := filepath.Join(t.TempDir(), "x.sql")
			require.NoError(t, sut.WriteFile(p, []byte("SELECT 1;"), 0o600))

			b, err := sut.ReadFile(p)
			require.NoError(t, err)
			assert.Equal(t, "SELECT 1;", string(b))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないファイルはエラー", func(t *testing.T) {
			t.Parallel()
			_, err := sut.ReadFile(filepath.Join(t.TempDir(), "missing.sql"))
			require.Error(t, err)
		})
	})
}

func TestOSFileSystem_WriteFile(t *testing.T) {
	t.Parallel()

	var sut osFileSystem

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定パスへ内容を書き込む", func(t *testing.T) {
			t.Parallel()
			p := filepath.Join(t.TempDir(), "x.sql")
			require.NoError(t, sut.WriteFile(p, []byte("SELECT 1;"), 0o600))

			b, err := os.ReadFile(p) //nolint:gosec // テスト内で生成したパスのみ読み込む
			require.NoError(t, err)
			assert.Equal(t, "SELECT 1;", string(b))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しない親ディレクトリへの書き込みはエラー", func(t *testing.T) {
			t.Parallel()
			p := filepath.Join(t.TempDir(), "missing", "x.sql")
			require.Error(t, sut.WriteFile(p, []byte("x"), 0o600))
		})
	})
}

func TestOSFileSystem_Remove(t *testing.T) {
	t.Parallel()

	var sut osFileSystem

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ファイルを削除できる", func(t *testing.T) {
			t.Parallel()
			p := filepath.Join(t.TempDir(), "x.sql")
			require.NoError(t, os.WriteFile(p, []byte("x"), 0o600))
			require.NoError(t, sut.Remove(p))

			_, err := os.Stat(p)
			require.Error(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないファイルはエラー", func(t *testing.T) {
			t.Parallel()
			require.Error(t, sut.Remove(filepath.Join(t.TempDir(), "missing.sql")))
		})
	})
}
