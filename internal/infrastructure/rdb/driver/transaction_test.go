package driver

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	mock_driver "go-boilerplate/internal/infrastructure/rdb/driver/mock"
	"go-boilerplate/internal/infrastructure/system"
	"go-boilerplate/internal/logging"
)

// recordingSleeper は、Sleep 呼び出し回数を記録し即時に返すテスト用 sleeper です。
type recordingSleeper struct{ calls int }

func (s *recordingSleeper) Sleep(context.Context, time.Duration) error { s.calls++; return nil }

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
	require.NotNil(t, manager)
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
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("トランザクションがロールバックされる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			err := manager.Do(ctx, func(_ context.Context) error {
				return errors.New("rollback")
			})
			// H1: fn が返す非 DB エラー（pg / 接続でない）は正規化せず生のまま返す（既存契約の維持）。
			require.Error(t, err)
			require.EqualError(t, err, "rollback")
		})

		t.Run("パニックが発生した場合にロールバックされる", func(t *testing.T) {
			t.Parallel()

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
}

func TestTxManager_Do_Retry(t *testing.T) {
	t.Parallel()

	// Begin 経路で生 SQLSTATE を注入し、pgx.Tx を mock せずにリトライ挙動を検証する。
	retryablePgErr := &pgconn.PgError{Code: "40001"}    // serialization_failure
	nonRetryablePgErr := &pgconn.PgError{Code: "23505"} // unique_violation

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("serialization_failureが続くとmaxAttemptsまで再試行しErrUnavailableを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			db.EXPECT().Begin(gomock.Any()).Return(nil, retryablePgErr).Times(defaultTxMaxAttempts)
			sleeper := &recordingSleeper{}

			dbCfg := config.NewDatabaseConfig(config.MockConfigForTest(t))
			m := NewTransactionManager(db, dbCfg, logging.NewTestLogger(t), sleeper)
			err := m.Do(context.Background(), func(context.Context) error { return nil })

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			// 試行間の sleep は maxAttempts-1 回。
			assert.Equal(t, defaultTxMaxAttempts-1, sleeper.calls)
		})

		t.Run("リトライ不可エラーは1回で返し待機しない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			db.EXPECT().Begin(gomock.Any()).Return(nil, nonRetryablePgErr).Times(1)
			sleeper := &recordingSleeper{}

			dbCfg := config.NewDatabaseConfig(config.MockConfigForTest(t))
			m := NewTransactionManager(db, dbCfg, logging.NewTestLogger(t), sleeper)
			err := m.Do(context.Background(), func(context.Context) error { return nil })

			require.Error(t, err)
			assert.Equal(t, 0, sleeper.calls)
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
