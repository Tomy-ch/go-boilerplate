package rdbdriver

import (
	"context"
	"database/sql"
	"testing"

	"boilerplate-go/internal/config"

	"github.com/stretchr/testify/require"
)

func TestResolveConn(t *testing.T) {
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOSConfig(cfg)
	dbCfg.SetDatabaseHost(t, "localhost")

	db, err := sql.Open("pgx", dbCfg.DatabaseDSN(osCfg))
	require.NoError(t, err)
	t.Cleanup(func() {
		err := db.Close()
		require.NoError(t, err)
	})

	t.Run("トランザクションが存在する場合", func(t *testing.T) {
		tx, err := db.BeginTx(context.Background(), nil)
		require.NoError(t, err)
		t.Cleanup(func() {
			err := tx.Rollback()
			require.NoError(t, err)
		})

		ctx := withTx(context.Background(), tx)
		conn := ResolveDriver(ctx, db)
		require.Equal(t, tx, conn)
	})

	t.Run("トランザクションが存在しない場合", func(t *testing.T) {
		ctx := context.Background()
		conn := ResolveDriver(ctx, db)
		require.Equal(t, db, conn)
	})
}
