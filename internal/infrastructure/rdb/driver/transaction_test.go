package driver

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/system"
	"go-boilerplate/internal/logging"
)

func TestNewTransactionManager(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)
	osCfg := config.NewOperatingSystemConfig(cfg)
	testLogger := logging.NewTestLogger(t)

	db, err := NewDB(dbCfg, osCfg, dbConnCfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	manager := NewTransactionManager(db, dbCfg, testLogger, system.NewSleeper())
	assert.NotNil(t, manager)
}

func TestTxManager_Do(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)
	osCfg := config.NewOperatingSystemConfig(cfg)
	dbCfg.SetDatabaseHost(t, "localhost")

	testLogger := logging.NewTestLogger(t)

	db, err := NewDB(dbCfg, osCfg, dbConnCfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	manager := NewTransactionManager(db, dbCfg, testLogger, system.NewSleeper())
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("トランザクションがコミットされる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			err := manager.Do(ctx, func(_ context.Context) error {
				return nil
			})
			require.NoError(t, err)
		})

		t.Run("すでにtxがある場合", func(t *testing.T) {
			t.Parallel()

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

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("トランザクションがロールバックされる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			err := manager.Do(ctx, func(_ context.Context) error {
				return errors.New("rollback")
			})
			// fn が返す非 DB エラー（pg / 接続でない）は正規化せず生のまま返す（既存契約の維持）。
			require.Error(t, err)
			require.EqualError(t, err, "rollback")
		})

		t.Run("パニックが発生した場合にロールバックされる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			defer func() {
				r := recover()
				assert.NotNil(t, r)
			}()

			_ = manager.Do(ctx, func(_ context.Context) error {
				panic("panic occurred")
			})
		})
	})
}

func TestTxManager_Do_Goexit(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)
	osCfg := config.NewOperatingSystemConfig(cfg)
	dbCfg.SetDatabaseHost(t, "localhost")
	testLogger := logging.NewTestLogger(t)

	db, err := NewDB(dbCfg, osCfg, dbConnCfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	manager := NewTransactionManager(db, dbCfg, testLogger, system.NewSleeper())

	// fn が runtime.Goexit（testify の FailNow と同じ中断）で抜けても
	// ロールバックされ、取得済み接続がプールへ返却される（リークしない）こと。
	before := db.Stats().AcquiredConns()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = manager.Do(context.Background(), func(_ context.Context) error {
			runtime.Goexit()
			return nil
		})
	}()
	<-done

	require.Eventually(t, func() bool {
		return db.Stats().AcquiredConns() <= before
	}, 2*time.Second, 10*time.Millisecond)
}
