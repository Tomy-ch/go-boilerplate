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
	"github.com/stretchr/testify/assert"
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

			require.NoError(t, execSeedFile(context.Background(), fsys, db, logging.NewTestLogger(t), nil, path))
		})

		t.Run("対象テーブル未作成はスキップしてnilを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			fsys.EXPECT().ReadFile(path).Return([]byte("SELECT 1;"), nil)
			db.EXPECT().Exec(gomock.Any(), gomock.Any()).Return(pgconn.CommandTag{}, &pgconn.PgError{Code: relationDoesNotExistCode})

			require.NoError(t, execSeedFile(context.Background(), fsys, db, logging.NewTestLogger(t), nil, path))
		})

		t.Run("プレースホルダを展開したSQLを実行する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			vars := map[string]string{"AUTH_ISSUER": "http://localhost:4003"}
			fsys.EXPECT().ReadFile(path).Return([]byte("INSERT ... VALUES ('${AUTH_ISSUER}');"), nil)
			db.EXPECT().Exec(gomock.Any(), "INSERT ... VALUES ('http://localhost:4003');").Return(pgconn.CommandTag{}, nil)

			require.NoError(t, execSeedFile(context.Background(), fsys, db, logging.NewTestLogger(t), vars, path))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("値の無いプレースホルダが残る場合はSQLを実行せずエラーを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			fsys.EXPECT().ReadFile(path).Return([]byte("INSERT ... VALUES ('${AUTH_ISSUER}');"), nil)

			err := execSeedFile(context.Background(), fsys, db, logging.NewTestLogger(t), nil, path)
			require.ErrorIs(t, err, errUndefinedPlaceholder)
		})

		t.Run("SQLリテラルを抜け出せる値はSQLを実行せずエラーを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			vars := map[string]string{"AUTH_ISSUER": "'); DROP TABLE user_identities; --"}
			fsys.EXPECT().ReadFile(path).Return([]byte("INSERT ... VALUES ('${AUTH_ISSUER}');"), nil)

			err := execSeedFile(context.Background(), fsys, db, logging.NewTestLogger(t), vars, path)
			require.ErrorIs(t, err, errUnsafePlaceholderValue)
		})

		t.Run("ファイル読み込み失敗はエラーを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			fsys.EXPECT().ReadFile(path).Return(nil, xerrors.New("read failed"))

			err := execSeedFile(context.Background(), fsys, db, logging.NewTestLogger(t), nil, path)
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

			err := execSeedFile(context.Background(), fsys, db, logging.NewTestLogger(t), nil, path)
			require.ErrorIs(t, err, execErr)
		})
	})
}

func Test_expandPlaceholders(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同一プレースホルダの全出現を値へ置き換える", func(t *testing.T) {
			t.Parallel()
			vars := map[string]string{"AUTH_ISSUER": "http://localhost:4003"}

			got, err := expandPlaceholders("('${AUTH_ISSUER}'),('${AUTH_ISSUER}');", vars)
			require.NoError(t, err)
			assert.Equal(t, "('http://localhost:4003'),('http://localhost:4003');", got)
		})

		t.Run("プレースホルダの無いSQLは値を渡さなくてもそのまま返る", func(t *testing.T) {
			t.Parallel()

			got, err := expandPlaceholders("SELECT 1;", nil)
			require.NoError(t, err)
			assert.Equal(t, "SELECT 1;", got)
		})

		t.Run("プレースホルダ以外のドル記号は置き換えない", func(t *testing.T) {
			t.Parallel()

			got, err := expandPlaceholders("SELECT $1, $$body$$;", nil)
			require.NoError(t, err)
			assert.Equal(t, "SELECT $1, $$body$$;", got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("値が渡されないプレースホルダはエラーになる", func(t *testing.T) {
			t.Parallel()

			_, err := expandPlaceholders("('${AUTH_ISSUER}');", nil)
			require.ErrorIs(t, err, errUndefinedPlaceholder)
			assert.Contains(t, err.Error(), "AUTH_ISSUER")
		})

		t.Run("空文字の値は空文字で埋めずエラーになる", func(t *testing.T) {
			t.Parallel()
			vars := map[string]string{"AUTH_ISSUER": ""}

			_, err := expandPlaceholders("('${AUTH_ISSUER}');", vars)
			require.ErrorIs(t, err, errUndefinedPlaceholder)
		})

		t.Run("単一引用符を含む値は展開せずエラーになる", func(t *testing.T) {
			t.Parallel()
			vars := map[string]string{"AUTH_ISSUER": "'); DROP TABLE user_identities; --"}

			_, err := expandPlaceholders("('${AUTH_ISSUER}');", vars)
			require.ErrorIs(t, err, errUnsafePlaceholderValue)
			assert.Contains(t, err.Error(), "AUTH_ISSUER")
		})

		t.Run("値が無いものと危険なものが混在する場合は値の無い方を報告する", func(t *testing.T) {
			t.Parallel()
			vars := map[string]string{"UNSAFE": "it's"}

			_, err := expandPlaceholders("('${AUTH_ISSUER}','${UNSAFE}');", vars)
			require.ErrorIs(t, err, errUndefinedPlaceholder)
		})

		t.Run("値の無いプレースホルダ名を全て列挙する", func(t *testing.T) {
			t.Parallel()
			vars := map[string]string{"KNOWN": "value"}

			_, err := expandPlaceholders("('${AUTH_ISSUER}','${KNOWN}','${OTHER}');", vars)
			require.ErrorIs(t, err, errUndefinedPlaceholder)
			assert.Contains(t, err.Error(), "AUTH_ISSUER, OTHER")
		})
	})
}

func Test_sortedNames(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("map の反復順に依らず昇順で連結する", func(t *testing.T) {
			t.Parallel()
			names := map[string]struct{}{"B": {}, "A": {}, "C": {}}

			assert.Equal(t, "A, B, C", sortedNames(names))
		})

		t.Run("空の集合は空文字を返す", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, sortedNames(map[string]struct{}{}))
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

			err := runSeeds(context.Background(), fsys, db, logging.NewTestLogger(t), nil, []string{"b.sql", "a.sql"})
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

			err := runSeeds(context.Background(), fsys, db, logging.NewTestLogger(t), nil, []string{"a.sql", "b.sql"})
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
			err := RunDBSeed(logging.NewTestLogger(t), fsys, "local", nil, openDB)
			require.NoError(t, err)
		})

		t.Run("渡したプレースホルダの値が実行するSQLへ反映される", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			fsys := mock_fs.NewMockFS(ctrl)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			vars := map[string]string{"AUTH_ISSUER": "http://localhost:4003"}

			fsys.EXPECT().Glob(seedFilePlace+"/*.sql").Return([]string{"a.sql"}, nil)
			fsys.EXPECT().ReadFile("a.sql").Return([]byte("INSERT ... VALUES ('${AUTH_ISSUER}');"), nil)
			db.EXPECT().Exec(gomock.Any(), "INSERT ... VALUES ('http://localhost:4003');").Return(pgconn.CommandTag{}, nil)
			db.EXPECT().Close().Return(nil)

			openDB := func(_ logging.Logger, _ string) (driver.DatabaseDriver, error) { return db, nil }
			err := RunDBSeed(logging.NewTestLogger(t), fsys, "local", vars, openDB)
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
			err := RunDBSeed(logging.NewTestLogger(t), fsys, "local", nil, openDB)
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
			err := RunDBSeed(logging.NewTestLogger(t), fsys, "local", nil, openDB)
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
			err := RunDBSeed(logging.NewTestLogger(t), fsys, "local", nil, openDB)
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
			err := RunDBSeed(logging.NewTestLogger(t), fsys, "local", nil, openDB)
			require.ErrorIs(t, err, execErr)
		})
	})
}
