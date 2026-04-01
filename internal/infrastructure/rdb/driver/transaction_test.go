package driver

import (
	"context"
	"errors"
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/logging"

	"github.com/stretchr/testify/require"
)

func TestNewTransactionManager(t *testing.T) {
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)
	testLogger := logging.NewTestLogger(t)

	db, err := NewDB(dbCfg, osCfg, dbConnCfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	manager := NewTransactionManager(cfg, db, testLogger)
	require.NotNil(t, manager)
}

func TestTxManager_Do(t *testing.T) {
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)
	dbCfg.SetDatabaseHost(t, "localhost")

	testLogger := logging.NewTestLogger(t)

	db, err := NewDB(dbCfg, osCfg, dbConnCfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	manager := NewTransactionManager(cfg, db, testLogger)
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("トランザクションがコミットされる", func(t *testing.T) {
			ctx := context.Background()
			err := manager.Do(ctx, func(_ context.Context) error {
				return nil
			})
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("トランザクションがロールバックされる", func(t *testing.T) {
			ctx := context.Background()
			err := manager.Do(ctx, func(_ context.Context) error {
				return errors.New("rollback")
			})
			require.Error(t, err)
			require.EqualError(t, err, "rollback")
		})

		t.Run("パニックが発生した場合にロールバックされる", func(t *testing.T) {
			ctx := context.Background()
			defer func() {
				r := recover()
				require.NotNil(t, r)
			}()

			_ = manager.Do(ctx, func(_ context.Context) error {
				panic("panic occurred")
			})
		})

		t.Run("すでにtxがある場合", func(t *testing.T) {
			ctx := context.Background()
			tx, err := db.Begin(ctx)
			require.NoError(t, err)
			t.Cleanup(func() {
				err = tx.Rollback(context.Background())
				require.NoError(t, err)
			})

			ctx = withTx(ctx, tx)
			err = manager.Do(ctx, func(_ context.Context) error {
				return nil
			})
			require.NoError(t, err)
		})
	})
}
