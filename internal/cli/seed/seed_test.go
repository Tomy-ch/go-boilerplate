package seed

import (
	"context"
	"testing"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	mock_driver "go-boilerplate/internal/infrastructure/rdb/driver/mock"
	"go-boilerplate/internal/logging"
	mock_fs "go-boilerplate/pkg/fs/mock"
	"go-boilerplate/pkg/xerrors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_handleSeedExecResult(t *testing.T) {
	t.Parallel()

	otherErr := xerrors.New("boom")
	pgRelationNotExist := &pgconn.PgError{Code: relationDoesNotExistCode}
	pgSyntaxErr := &pgconn.PgError{Code: "42601"} // syntax_error

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("実行成功はnilを返す", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, handleSeedExecResult(context.Background(), logging.NewTestLogger(t), "seed.sql", nil))
		})
		t.Run("対象テーブル未作成はスキップしてnilを返す", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, handleSeedExecResult(context.Background(), logging.NewTestLogger(t), "seed.sql", pgRelationNotExist))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("テーブル未作成以外のPostgreSQLエラーは伝播する", func(t *testing.T) {
			t.Parallel()
			err := handleSeedExecResult(context.Background(), logging.NewTestLogger(t), "seed.sql", pgSyntaxErr)
			require.ErrorIs(t, err, pgSyntaxErr)
		})
		t.Run("PostgreSQL以外のエラーも伝播する", func(t *testing.T) {
			t.Parallel()
			err := handleSeedExecResult(context.Background(), logging.NewTestLogger(t), "seed.sql", otherErr)
			require.ErrorIs(t, err, otherErr)
		})
	})
}

func Test_execSeedFile(t *testing.T) {
	t.Parallel()

	const path = "database/seed/001.sql"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("読み込みと実行に成功するとnilを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			fsys.EXPECT().ReadFile(path).Return([]byte("SELECT 1;"), nil)
			db.EXPECT().Exec(gomock.Any(), "SELECT 1;").Return(pgconn.CommandTag{}, nil)

			require.NoError(t, execSeedFile(context.Background(), fsys, db, logging.NewTestLogger(t), path))
		})

		t.Run("対象テーブル未作成はスキップしてnilを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			fsys.EXPECT().ReadFile(path).Return([]byte("SELECT 1;"), nil)
			db.EXPECT().Exec(gomock.Any(), gomock.Any()).Return(pgconn.CommandTag{}, &pgconn.PgError{Code: relationDoesNotExistCode})

			require.NoError(t, execSeedFile(context.Background(), fsys, db, logging.NewTestLogger(t), path))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ファイル読み込み失敗はエラーを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			fsys.EXPECT().ReadFile(path).Return(nil, xerrors.New("read failed"))

			err := execSeedFile(context.Background(), fsys, db, logging.NewTestLogger(t), path)
			require.Error(t, err)
		})

		t.Run("実SQLエラーは握り潰さず伝播する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			execErr := &pgconn.PgError{Code: "23505"} // unique_violation
			fsys.EXPECT().ReadFile(path).Return([]byte("INSERT ..."), nil)
			db.EXPECT().Exec(gomock.Any(), gomock.Any()).Return(pgconn.CommandTag{}, execErr)

			err := execSeedFile(context.Background(), fsys, db, logging.NewTestLogger(t), path)
			require.ErrorIs(t, err, execErr)
		})
	})
}

func Test_runSeeds(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("逆順で渡しても昇順でExecが呼ばれることを順序固定で検証する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)

			// 内部 sort.Strings の昇順実行を gomock.InOrder で固定し、
			// a.sql → b.sql の順で ReadFile と Exec が走ることを検証する。
			gomock.InOrder(
				fsys.EXPECT().ReadFile("a.sql").Return([]byte("SELECT a;"), nil),
				db.EXPECT().Exec(gomock.Any(), "SELECT a;").Return(pgconn.CommandTag{}, nil),
				fsys.EXPECT().ReadFile("b.sql").Return([]byte("SELECT b;"), nil),
				db.EXPECT().Exec(gomock.Any(), "SELECT b;").Return(pgconn.CommandTag{}, nil),
			)

			err := runSeeds(context.Background(), fsys, db, logging.NewTestLogger(t), []string{"b.sql", "a.sql"})
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("1ファイルが失敗しても全件継続しエラーを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			execErr := &pgconn.PgError{Code: "23505"}
			fsys.EXPECT().ReadFile("a.sql").Return([]byte("SELECT a;"), nil)
			fsys.EXPECT().ReadFile("b.sql").Return([]byte("SELECT b;"), nil)
			db.EXPECT().Exec(gomock.Any(), "SELECT a;").Return(pgconn.CommandTag{}, execErr)
			db.EXPECT().Exec(gomock.Any(), "SELECT b;").Return(pgconn.CommandTag{}, nil)

			err := runSeeds(context.Background(), fsys, db, logging.NewTestLogger(t), []string{"a.sql", "b.sql"})
			require.ErrorIs(t, err, execErr)
		})
	})
}

func TestRunDBSeed(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DB接続後にseedファイルを列挙し投入する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)

			fsys.EXPECT().Glob(seedFilePlace+"/*.sql").Return([]string{"a.sql"}, nil)
			fsys.EXPECT().ReadFile("a.sql").Return([]byte("SELECT 1;"), nil)
			db.EXPECT().Exec(gomock.Any(), "SELECT 1;").Return(pgconn.CommandTag{}, nil)
			db.EXPECT().Close().Return(nil)

			openDB := func(_ logging.Logger, _ string) (driver.DatabaseDriver, error) { return db, nil }
			err := RunDBSeed(logging.NewTestLogger(t), fsys, "local", openDB)
			require.NoError(t, err)
		})

		t.Run("Closeが失敗してもログのみで投入結果を優先する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)

			fsys.EXPECT().Glob(seedFilePlace+"/*.sql").Return([]string{}, nil)
			db.EXPECT().Close().Return(xerrors.New("close failed"))

			openDB := func(_ logging.Logger, _ string) (driver.DatabaseDriver, error) { return db, nil }
			err := RunDBSeed(logging.NewTestLogger(t), fsys, "local", openDB)
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DB接続に失敗するとエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)

			openDB := func(_ logging.Logger, _ string) (driver.DatabaseDriver, error) {
				return nil, xerrors.New("open failed")
			}
			err := RunDBSeed(logging.NewTestLogger(t), fsys, "local", openDB)
			require.Error(t, err)
		})

		t.Run("seedファイルの列挙に失敗するとエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)

			fsys.EXPECT().Glob(seedFilePlace+"/*.sql").Return(nil, xerrors.New("glob failed"))
			// 接続は確立済みのため Close は必ず呼ばれる。
			db.EXPECT().Close().Return(nil)

			openDB := func(_ logging.Logger, _ string) (driver.DatabaseDriver, error) { return db, nil }
			err := RunDBSeed(logging.NewTestLogger(t), fsys, "local", openDB)
			require.Error(t, err)
		})

		t.Run("seed投入自体が失敗するとrunSeedsのエラーが伝播する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)

			execErr := &pgconn.PgError{Code: "23505"} // unique_violation
			fsys.EXPECT().Glob(seedFilePlace+"/*.sql").Return([]string{"a.sql"}, nil)
			fsys.EXPECT().ReadFile("a.sql").Return([]byte("INSERT ..."), nil)
			db.EXPECT().Exec(gomock.Any(), "INSERT ...").Return(pgconn.CommandTag{}, execErr)
			db.EXPECT().Close().Return(nil)

			openDB := func(_ logging.Logger, _ string) (driver.DatabaseDriver, error) { return db, nil }
			err := RunDBSeed(logging.NewTestLogger(t), fsys, "local", openDB)
			require.ErrorIs(t, err, execErr)
		})
	})
}
