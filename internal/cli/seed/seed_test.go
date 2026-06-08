package seed

import (
	"context"
	"errors"
	"testing"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	mock_driver "go-boilerplate/internal/infrastructure/rdb/driver/mock"
	"go-boilerplate/internal/logging"
	mock_fs "go-boilerplate/pkg/fs/mock"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandleSeedExecResult(t *testing.T) {
	t.Parallel()

	otherErr := errors.New("boom")
	pgRelationNotExist := &pgconn.PgError{Code: relationDoesNotExistCode}
	pgSyntaxErr := &pgconn.PgError{Code: "42601"} // syntax_error

	tests := []struct {
		name    string
		execErr error
		wantErr error // nil ならエラーを返さない（成功 or スキップ）
	}{
		{name: "正常系_実行成功はnilを返す", execErr: nil, wantErr: nil},
		{name: "正常系_対象テーブル未作成はスキップしてnilを返す", execErr: pgRelationNotExist, wantErr: nil},
		{name: "異常系_テーブル未作成以外のPostgreSQLエラーは伝播する", execErr: pgSyntaxErr, wantErr: pgSyntaxErr},
		{name: "異常系_PostgreSQL以外のエラーも伝播する", execErr: otherErr, wantErr: otherErr},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := handleSeedExecResult(logging.NewTestLogger(t), "seed.sql", tt.execErr)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestExecSeedFile(t *testing.T) {
	t.Parallel()

	const path = "database/seed/001.sql"

	t.Run("正常系_読み込みと実行に成功するとnilを返す", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fsys := mock_fs.NewMockFS(ctrl)
		db := mock_driver.NewMockDatabaseDriver(ctrl)
		fsys.EXPECT().ReadFile(path).Return([]byte("SELECT 1;"), nil)
		db.EXPECT().Exec(gomock.Any(), "SELECT 1;").Return(pgconn.CommandTag{}, nil)

		require.NoError(t, execSeedFile(context.Background(), fsys, db, logging.NewTestLogger(t), path))
	})

	t.Run("異常系_ファイル読み込み失敗はエラーを返す", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fsys := mock_fs.NewMockFS(ctrl)
		db := mock_driver.NewMockDatabaseDriver(ctrl)
		fsys.EXPECT().ReadFile(path).Return(nil, errors.New("read failed"))

		err := execSeedFile(context.Background(), fsys, db, logging.NewTestLogger(t), path)
		require.Error(t, err)
	})

	t.Run("異常系_実SQLエラーは握り潰さず伝播する", func(t *testing.T) {
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

	t.Run("正常系_対象テーブル未作成はスキップしてnilを返す", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fsys := mock_fs.NewMockFS(ctrl)
		db := mock_driver.NewMockDatabaseDriver(ctrl)
		fsys.EXPECT().ReadFile(path).Return([]byte("SELECT 1;"), nil)
		db.EXPECT().Exec(gomock.Any(), gomock.Any()).Return(pgconn.CommandTag{}, &pgconn.PgError{Code: relationDoesNotExistCode})

		require.NoError(t, execSeedFile(context.Background(), fsys, db, logging.NewTestLogger(t), path))
	})
}

func TestRunSeeds(t *testing.T) {
	t.Parallel()

	t.Run("正常系_全ファイル成功時はnilを返す", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fsys := mock_fs.NewMockFS(ctrl)
		db := mock_driver.NewMockDatabaseDriver(ctrl)
		fsys.EXPECT().ReadFile("a.sql").Return([]byte("SELECT 1;"), nil)
		fsys.EXPECT().ReadFile("b.sql").Return([]byte("SELECT 2;"), nil)
		db.EXPECT().Exec(gomock.Any(), gomock.Any()).Return(pgconn.CommandTag{}, nil).Times(2)

		err := runSeeds(context.Background(), fsys, db, logging.NewTestLogger(t), []string{"b.sql", "a.sql"})
		require.NoError(t, err)
	})

	t.Run("異常系_1ファイルが失敗しても全件継続しエラーを返す", func(t *testing.T) {
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
}

func TestRunDBSeed(t *testing.T) {
	t.Parallel()

	t.Run("正常系_DB接続後にseedファイルを列挙し投入する", func(t *testing.T) {
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

	t.Run("異常系_DB接続に失敗するとエラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fsys := mock_fs.NewMockFS(ctrl)

		openDB := func(_ logging.Logger, _ string) (driver.DatabaseDriver, error) {
			return nil, errors.New("open failed")
		}
		err := RunDBSeed(logging.NewTestLogger(t), fsys, "local", openDB)
		require.Error(t, err)
	})

	t.Run("異常系_seedファイルの列挙に失敗するとエラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fsys := mock_fs.NewMockFS(ctrl)
		db := mock_driver.NewMockDatabaseDriver(ctrl)

		fsys.EXPECT().Glob(seedFilePlace+"/*.sql").Return(nil, errors.New("glob failed"))
		// 接続は確立済みのため Close は必ず呼ばれる。
		db.EXPECT().Close().Return(nil)

		openDB := func(_ logging.Logger, _ string) (driver.DatabaseDriver, error) { return db, nil }
		err := RunDBSeed(logging.NewTestLogger(t), fsys, "local", openDB)
		require.Error(t, err)
	})

	t.Run("正常系_Closeが失敗してもログのみで投入結果を優先する", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		fsys := mock_fs.NewMockFS(ctrl)
		db := mock_driver.NewMockDatabaseDriver(ctrl)

		fsys.EXPECT().Glob(seedFilePlace+"/*.sql").Return([]string{}, nil)
		db.EXPECT().Close().Return(errors.New("close failed"))

		openDB := func(_ logging.Logger, _ string) (driver.DatabaseDriver, error) { return db, nil }
		err := RunDBSeed(logging.NewTestLogger(t), fsys, "local", openDB)
		require.NoError(t, err)
	})
}
