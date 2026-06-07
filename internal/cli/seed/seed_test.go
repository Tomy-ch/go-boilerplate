package seed

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	mock_driver "go-boilerplate/internal/infrastructure/rdb/driver/mock"
	"go-boilerplate/internal/logging"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
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

	t.Run("正常系_読み込みと実行に成功するとnilを返す", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		db := mock_driver.NewMockDatabaseDriver(ctrl)
		db.EXPECT().Exec(gomock.Any(), "SELECT 1;").Return(pgconn.CommandTag{}, nil)

		path := filepath.Join(t.TempDir(), "001.sql")
		require.NoError(t, os.WriteFile(path, []byte("SELECT 1;"), 0o600))

		require.NoError(t, execSeedFile(context.Background(), db, logging.NewTestLogger(t), path))
	})

	t.Run("異常系_ファイル読み込み失敗はエラーを返す", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		db := mock_driver.NewMockDatabaseDriver(ctrl)
		// Exec は呼ばれない。

		err := execSeedFile(context.Background(), db, logging.NewTestLogger(t), filepath.Join(t.TempDir(), "not-exist.sql"))
		require.Error(t, err)
	})

	t.Run("異常系_実SQLエラーは握り潰さず伝播する", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		db := mock_driver.NewMockDatabaseDriver(ctrl)
		execErr := &pgconn.PgError{Code: "23505"} // unique_violation
		db.EXPECT().Exec(gomock.Any(), gomock.Any()).Return(pgconn.CommandTag{}, execErr)

		path := filepath.Join(t.TempDir(), "001.sql")
		require.NoError(t, os.WriteFile(path, []byte("INSERT ..."), 0o600))

		err := execSeedFile(context.Background(), db, logging.NewTestLogger(t), path)
		require.ErrorIs(t, err, execErr)
	})

	t.Run("正常系_対象テーブル未作成はスキップしてnilを返す", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		db := mock_driver.NewMockDatabaseDriver(ctrl)
		db.EXPECT().Exec(gomock.Any(), gomock.Any()).Return(pgconn.CommandTag{}, &pgconn.PgError{Code: relationDoesNotExistCode})

		path := filepath.Join(t.TempDir(), "001.sql")
		require.NoError(t, os.WriteFile(path, []byte("SELECT 1;"), 0o600))

		require.NoError(t, execSeedFile(context.Background(), db, logging.NewTestLogger(t), path))
	})
}

func TestNewDBSeedCommand(t *testing.T) {
	t.Parallel()

	cmd := NewDBSeedCommand()
	assert.Equal(t, "db-seed", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("database"))
}
