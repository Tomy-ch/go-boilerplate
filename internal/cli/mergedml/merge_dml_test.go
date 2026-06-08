package mergedml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-boilerplate/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestGenerator は、テスト用に workDir を差し替えた generator を生成します。
func newTestGenerator(t *testing.T, workDir string) *generator {
	t.Helper()
	return &generator{
		logger:          logging.NewTestLogger(t),
		callerSkipCount: 1,
		workDir:         workDir,
		dmlRootDir:      "database/dml/",
		genRootDir:      "database/gen/",
		sqlcCfg:         "sqlc.yaml",
	}
}

// writeFile は、親ディレクトリを作成したうえでファイルを書き出すテストヘルパです。
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

// readFile は、テスト用の一時ファイルを読み出すヘルパです。
func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path) //nolint:gosec // G304: テスト用の一時ディレクトリ配下のパスで外部入力ではない
	require.NoError(t, err)
	return string(b)
}

func TestGenerator_buildCategorySQLFile(t *testing.T) {
	t.Parallel()

	t.Run("正常系_workDir起点で入力を解決しCWDに依存せず連結する", func(t *testing.T) {
		t.Parallel()
		// workDir をテンポラリに設定する。CWD（パッケージディレクトリ）には一切ファイルを作らず、
		// workDir 配下にのみ入力を置くことで、パス解決が workDir 起点であることを検証する。
		work := t.TempDir()
		writeFile(t, filepath.Join(work, "database/dml/repository/user/001_first.sql"), "SELECT 1;")
		writeFile(t, filepath.Join(work, "database/dml/repository/user/002_second.sql"), "SELECT 2;")
		require.NoError(t, os.MkdirAll(filepath.Join(work, "database/gen"), 0o750))

		g := newTestGenerator(t, work)
		require.NoError(t, g.buildCategorySQLFile("user", "repository"))

		got := readFile(t, filepath.Join(work, "database/gen/user_repository.gen.sql"))
		assert.Contains(t, got, "SELECT 1;")
		assert.Contains(t, got, "SELECT 2;")
		// ソート順（001 → 002）で連結されること。
		assert.Less(t, strings.Index(got, "SELECT 1;"), strings.Index(got, "SELECT 2;"))
		// 由来コメントが付与されること。
		assert.Contains(t, got, "-- === source:")
	})

	t.Run("正常系_SQLが空のカテゴリは生成物を削除する", func(t *testing.T) {
		t.Parallel()
		work := t.TempDir()
		// カテゴリディレクトリは存在するが .sql は無い状態を作る。
		require.NoError(t, os.MkdirAll(filepath.Join(work, "database/dml/repository/empty"), 0o750))
		// 旧生成物を事前に配置し、削除されることを確認する。
		dst := filepath.Join(work, "database/gen/empty_repository.gen.sql")
		writeFile(t, dst, "-- stale")

		g := newTestGenerator(t, work)
		require.NoError(t, g.buildCategorySQLFile("empty", "repository"))

		_, err := os.Stat(dst)
		assert.True(t, os.IsNotExist(err), "空カテゴリでは生成物が削除されるべき")
	})

	t.Run("正常系_末尾に改行が無いファイルでも連結が壊れない", func(t *testing.T) {
		t.Parallel()
		work := t.TempDir()
		writeFile(t, filepath.Join(work, "database/dml/repository/user/001.sql"), "SELECT 1;") // 末尾改行なし
		writeFile(t, filepath.Join(work, "database/dml/repository/user/002.sql"), "SELECT 2;")
		require.NoError(t, os.MkdirAll(filepath.Join(work, "database/gen"), 0o750))

		g := newTestGenerator(t, work)
		require.NoError(t, g.buildCategorySQLFile("user", "repository"))

		got := readFile(t, filepath.Join(work, "database/gen/user_repository.gen.sql"))
		// 1本目と2本目の SQL が改行で分離されていること（連結で "SELECT 1;SELECT 2;" にならない）。
		assert.NotContains(t, got, "SELECT 1;SELECT 2;")
	})
}

func TestGenerator_cleanupStaleGeneratedFiles(t *testing.T) {
	t.Parallel()

	// genDir は workDir 配下に database/gen を作り、与えたファイル群を配置するヘルパ。
	setup := func(t *testing.T, files ...string) (*generator, string) {
		t.Helper()
		work := t.TempDir()
		genDir := filepath.Join(work, "database/gen")
		require.NoError(t, os.MkdirAll(genDir, 0o750))
		for _, name := range files {
			require.NoError(t, os.WriteFile(filepath.Join(genDir, name), []byte("-- x"), 0o600))
		}
		return newTestGenerator(t, work), genDir
	}

	exists := func(t *testing.T, genDir, name string) bool {
		t.Helper()
		_, err := os.Stat(filepath.Join(genDir, name))
		return err == nil
	}

	t.Run("正常系_keep対象は残し同typeのstaleのみ削除し他typeは触らない", func(t *testing.T) {
		t.Parallel()
		g, genDir := setup(t,
			"user_repository.gen.sql",   // keep（今回の生成対象）
			"old_repository.gen.sql",    // stale（同 type・keep外）→削除
			"foo_query_service.gen.sql", // 別 type →非対象で温存
		)

		require.NoError(t, g.cleanupStaleGeneratedFiles([]string{"user"}, "repository"))

		assert.True(t, exists(t, genDir, "user_repository.gen.sql"), "keep 対象は残る")
		assert.False(t, exists(t, genDir, "old_repository.gen.sql"), "同 type の stale は削除される")
		assert.True(t, exists(t, genDir, "foo_query_service.gen.sql"), "別 type の生成物は触らない")
	})

	t.Run("正常系_カテゴリ0件のとき同typeの生成物を全削除し他typeは温存する", func(t *testing.T) {
		t.Parallel()
		g, genDir := setup(t,
			"a_repository.gen.sql",
			"b_repository.gen.sql",
			"x_query_service.gen.sql",
		)

		// categories が空＝このフローでは「全 stale 削除（同 type 全消し）」になる破壊的経路。
		require.NoError(t, g.cleanupStaleGeneratedFiles(nil, "repository"))

		assert.False(t, exists(t, genDir, "a_repository.gen.sql"))
		assert.False(t, exists(t, genDir, "b_repository.gen.sql"))
		assert.True(t, exists(t, genDir, "x_query_service.gen.sql"), "別 type は全消し対象外")
	})

	t.Run("異常系_genディレクトリが存在しない場合はエラー", func(t *testing.T) {
		t.Parallel()
		// gen ディレクトリを作らない workDir を渡すと os.ReadDir が失敗する。
		g := newTestGenerator(t, t.TempDir())
		err := g.cleanupStaleGeneratedFiles([]string{"user"}, "repository")
		require.Error(t, err)
	})
}

func TestGenerator_ensureUnderDir(t *testing.T) {
	t.Parallel()

	work := t.TempDir()
	g := newTestGenerator(t, work)

	t.Run("正常系_genRootDir配下のパスは許容される", func(t *testing.T) {
		t.Parallel()
		err := g.ensureUnderDir(filepath.Join(work, "database/gen", "user_repository.gen.sql"))
		require.NoError(t, err)
	})

	t.Run("異常系_genRootDirの外を指すパスはエラー", func(t *testing.T) {
		t.Parallel()
		err := g.ensureUnderDir(filepath.Join(work, "database", "outside.sql"))
		require.Error(t, err)
	})
}

func TestListDirs(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(base, "bbb"), 0o750))
	require.NoError(t, os.MkdirAll(filepath.Join(base, "aaa"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(base, "file.sql"), []byte("x"), 0o600))

	t.Run("正常系_サブディレクトリのみを昇順で返す", func(t *testing.T) {
		t.Parallel()
		dirs, err := listDirs(base)
		require.NoError(t, err)
		assert.Equal(t, []string{"aaa", "bbb"}, dirs)
	})

	t.Run("異常系_存在しないディレクトリはエラー", func(t *testing.T) {
		t.Parallel()
		_, err := listDirs(filepath.Join(base, "not-exist"))
		require.Error(t, err)
	})
}

func TestGenerator_dmlTypeRootAbs(t *testing.T) {
	t.Parallel()

	g := newTestGenerator(t, "/app")
	assert.Equal(t, filepath.Join("/app", "database/dml/", "repository"), g.dmlTypeRootAbs("repository"))
}

func TestResolveConcurrencyConst(t *testing.T) {
	t.Parallel()

	// 実行環境の NumCPU に依存するため、上限・下限の不変条件のみを検証する。
	got := resolveConcurrencyConst()
	assert.GreaterOrEqual(t, got, 1)
	assert.LessOrEqual(t, got, maxSQLCConcurrency)
}

func TestNewGenerator(t *testing.T) {
	t.Parallel()

	t.Cleanup(func() { workDir = "" })
	workDir = "/custom"
	g := newGenerator(logging.NewTestLogger(t))
	assert.Equal(t, "/custom", g.workDir)
	assert.Equal(t, "database/dml/", g.dmlRootDir)
	assert.Equal(t, "database/gen/", g.genRootDir)
	assert.Equal(t, 1, g.callerSkipCount)
}
