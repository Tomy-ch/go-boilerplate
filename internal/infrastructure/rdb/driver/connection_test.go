package driver

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)
	osCfg := config.NewOperatingSystemConfig(cfg)
	dbCfg.SetDatabaseHost(t, "localhost")

	db, err := NewDB(dbCfg, osCfg, dbConnCfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	t.Run("トランザクションが存在する場合", func(t *testing.T) {
		tx, err := db.Begin(context.Background())
		require.NoError(t, err)
		t.Cleanup(func() {
			err := tx.Rollback(context.Background())
			require.NoError(t, err)
		})

		ctx := withTx(context.Background(), tx)
		conn := New(ctx, db)
		assert.Equal(t, tx, conn)
	})

	t.Run("トランザクションが存在しない場合", func(t *testing.T) {
		ctx := context.Background()
		conn := New(ctx, db)
		assert.Equal(t, db, conn)
	})
}
