package driver_test

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
	"go-boilerplate/internal/infrastructure/rdb/driver"
	mock_driver "go-boilerplate/internal/infrastructure/rdb/driver/mock"
	"go-boilerplate/internal/infrastructure/system"
	"go-boilerplate/internal/logging"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
)

func TestTxManager_Do(t *testing.T) {
	t.Parallel()

	// Begin 経路で生 SQLSTATE を注入し、pgx.Tx を mock せずにリトライ挙動を検証する。
	retryablePgErr := &pgconn.PgError{Code: "40001"}    // serialization_failure
	nonRetryablePgErr := &pgconn.PgError{Code: "23505"} // unique_violation

	// 実 DB を共有する非モック系ケース用のマネージャ。
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)
	osCfg := config.NewOperatingSystemConfig(cfg)
	dbCfg.SetDatabaseHost(t, "localhost")

	db, err := driver.NewDB(dbCfg, osCfg, dbConnCfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	manager := driver.NewTransactionManager(db, dbCfg, logging.NewTestLogger(t), system.NewSleeper())

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

			ctx = driver.WithTx(ctx, tx)
			err = manager.Do(ctx, func(_ context.Context) error {
				return nil
			})
			require.NoError(t, err)
		})

		t.Run("1回目のリトライ可能エラー後に2回目の試行でコミットが成功する", func(t *testing.T) {
			t.Parallel()

			// pgx.Tx のモックが存在しないため Begin を mock で差し替える方式は採れない。
			// 実 DB を用い fn 内でカウンタにより1回目のみリトライ可能エラーを返す方式とする。
			cfg := config.MockConfigForTest(t)
			dbCfg := config.NewDatabaseConfig(cfg)
			dbConnCfg := config.NewDBConnectionConfig(cfg)
			osCfg := config.NewOperatingSystemConfig(cfg)
			dbCfg.SetDatabaseHost(t, "localhost")

			realDB, err := driver.NewDB(dbCfg, osCfg, dbConnCfg)
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, realDB.Close())
			})

			ctrl := gomock.NewController(t)
			sleeper := mock_clock.NewMockSleeper(ctrl)
			// 試行間の sleep は1回。
			sleeper.EXPECT().Sleep(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			m := driver.NewTransactionManager(realDB, dbCfg, logging.NewTestLogger(t), sleeper)

			attempts := 0
			err = m.Do(context.Background(), func(_ context.Context) error {
				attempts++
				if attempts == 1 {
					return retryablePgErr
				}
				return nil
			})

			require.NoError(t, err)
			assert.Equal(t, 2, attempts)
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

		t.Run("serialization_failureが続くとmaxAttemptsまで再試行しErrUnavailableを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockDB := mock_driver.NewMockDatabaseDriver(ctrl)
			dbCfg := config.NewDatabaseConfig(config.MockConfigForTest(t))
			// maxAttempts は config 値（DB_TX_MAX_RETRIES）から導出される。
			maxAttempts := dbCfg.TxMaxRetries()
			mockDB.EXPECT().Begin(gomock.Any()).Return(nil, retryablePgErr).Times(maxAttempts)
			sleeper := mock_clock.NewMockSleeper(ctrl)
			// 試行間の sleep は maxAttempts-1 回。
			sleeper.EXPECT().Sleep(gomock.Any(), gomock.Any()).Return(nil).Times(maxAttempts - 1)

			m := driver.NewTransactionManager(mockDB, dbCfg, logging.NewTestLogger(t), sleeper)
			err := m.Do(context.Background(), func(context.Context) error { return nil })

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("リトライ不可エラーは1回で返し待機しない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockDB := mock_driver.NewMockDatabaseDriver(ctrl)
			mockDB.EXPECT().Begin(gomock.Any()).Return(nil, nonRetryablePgErr).Times(1)
			sleeper := mock_clock.NewMockSleeper(ctrl)
			// リトライしないため待機は発生しない。
			sleeper.EXPECT().Sleep(gomock.Any(), gomock.Any()).Times(0)

			dbCfg := config.NewDatabaseConfig(config.MockConfigForTest(t))
			m := driver.NewTransactionManager(mockDB, dbCfg, logging.NewTestLogger(t), sleeper)
			err := m.Do(context.Background(), func(context.Context) error { return nil })

			require.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("リトライ待機中にcontextがキャンセルされると直前のエラーを正規化して返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockDB := mock_driver.NewMockDatabaseDriver(ctrl)
			// 1回目のリトライ可能エラー後、待機（Sleep）が ctx 打ち切りで失敗するため再試行されない。
			mockDB.EXPECT().Begin(gomock.Any()).Return(nil, retryablePgErr).Times(1)
			sleeper := mock_clock.NewMockSleeper(ctrl)
			sleeper.EXPECT().Sleep(gomock.Any(), gomock.Any()).Return(context.Canceled).Times(1)

			dbCfg := config.NewDatabaseConfig(config.MockConfigForTest(t))
			m := driver.NewTransactionManager(mockDB, dbCfg, logging.NewTestLogger(t), sleeper)
			err := m.Do(context.Background(), func(context.Context) error { return nil })

			// 待機の context.Canceled ではなく、リトライ対象だった元の失敗（40001）を
			// 正規化した ErrUnavailable が返る。
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("Beginがcontext.DeadlineExceededを返すとErrUnavailableへ正規化される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockDB := mock_driver.NewMockDatabaseDriver(ctrl)
			// context.DeadlineExceeded はリトライ対象外のため Begin は1回のみ。
			mockDB.EXPECT().Begin(gomock.Any()).Return(nil, context.DeadlineExceeded).Times(1)
			sleeper := mock_clock.NewMockSleeper(ctrl)
			// リトライ対象外のため待機は発生しない。
			sleeper.EXPECT().Sleep(gomock.Any(), gomock.Any()).Times(0)

			dbCfg := config.NewDatabaseConfig(config.MockConfigForTest(t))
			m := driver.NewTransactionManager(mockDB, dbCfg, logging.NewTestLogger(t), sleeper)
			err := m.Do(context.Background(), func(context.Context) error { return nil })

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("Beginがcontext.Canceledを返すとErrCanceledへ正規化される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockDB := mock_driver.NewMockDatabaseDriver(ctrl)
			mockDB.EXPECT().Begin(gomock.Any()).Return(nil, context.Canceled).Times(1)
			sleeper := mock_clock.NewMockSleeper(ctrl)
			// リトライ対象外のため待機は発生しない。
			sleeper.EXPECT().Sleep(gomock.Any(), gomock.Any()).Times(0)

			dbCfg := config.NewDatabaseConfig(config.MockConfigForTest(t))
			m := driver.NewTransactionManager(mockDB, dbCfg, logging.NewTestLogger(t), sleeper)
			err := m.Do(context.Background(), func(context.Context) error { return nil })

			require.ErrorIs(t, err, apperror.ErrCanceled)
		})

		t.Run("fnがruntime.Goexitで中断しても接続がリークしない", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			dbCfg := config.NewDatabaseConfig(cfg)
			dbConnCfg := config.NewDBConnectionConfig(cfg)
			osCfg := config.NewOperatingSystemConfig(cfg)
			dbCfg.SetDatabaseHost(t, "localhost")

			goexitDB, err := driver.NewDB(dbCfg, osCfg, dbConnCfg)
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, goexitDB.Close())
			})

			m := driver.NewTransactionManager(goexitDB, dbCfg, logging.NewTestLogger(t), system.NewSleeper())

			// fn が runtime.Goexit（testify の FailNow と同じ中断）で抜けても
			// ロールバックされ、取得済み接続がプールへ返却される（リークしない）こと。
			before := goexitDB.Stats().AcquiredConns()

			done := make(chan struct{})
			go func() {
				defer close(done)
				_ = m.Do(context.Background(), func(_ context.Context) error {
					runtime.Goexit()
					return nil
				})
			}()
			<-done

			require.Eventually(t, func() bool {
				return goexitDB.Stats().AcquiredConns() <= before
			}, 2*time.Second, 10*time.Millisecond)
		})
	})
}
