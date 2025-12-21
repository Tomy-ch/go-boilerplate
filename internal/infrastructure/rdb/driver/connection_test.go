package driver

import (
	"context"
	"database/sql"
	"testing"

	"boilerplate-go/internal/config"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOSConfig(cfg)
	dbCfg.SetDatabaseHost(t, "localhost")

	db, err := sql.Open("pgx", dbCfg.DSNWithTimeZone(osCfg))
	dbDriver := &dbDriver{db}
	require.NoError(t, err)
	t.Cleanup(func() {
		err := dbDriver.Close()
		require.NoError(t, err)
	})

	t.Run("トランザクションが存在する場合", func(t *testing.T) {
		tx, err := dbDriver.BeginTx(context.Background(), nil)
		require.NoError(t, err)
		t.Cleanup(func() {
			err := tx.Rollback()
			require.NoError(t, err)
		})

		ctx := withTx(context.Background(), tx)
		conn := New(ctx, dbDriver)
		require.Equal(t, tx, conn)
	})

	t.Run("トランザクションが存在しない場合", func(t *testing.T) {
		ctx := context.Background()
		conn := New(ctx, dbDriver)
		require.Equal(t, dbDriver, conn)
	})
}
