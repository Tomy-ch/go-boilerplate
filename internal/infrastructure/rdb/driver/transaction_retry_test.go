package driver_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	mock_driver "go-boilerplate/internal/infrastructure/rdb/driver/mock"
	"go-boilerplate/internal/logging"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
)

func TestTxManager_Do_Retry(t *testing.T) {
	t.Parallel()

	// Begin 経路で生 SQLSTATE を注入し、pgx.Tx を mock せずにリトライ挙動を検証する。
	retryablePgErr := &pgconn.PgError{Code: "40001"}    // serialization_failure
	nonRetryablePgErr := &pgconn.PgError{Code: "23505"} // unique_violation

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

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

		t.Run("serialization_failureが続くとmaxAttemptsまで再試行しErrUnavailableを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			dbCfg := config.NewDatabaseConfig(config.MockConfigForTest(t))
			// maxAttempts は config 値（DB_TX_MAX_RETRIES）から導出される。
			maxAttempts := dbCfg.TxMaxRetries()
			db.EXPECT().Begin(gomock.Any()).Return(nil, retryablePgErr).Times(maxAttempts)
			sleeper := mock_clock.NewMockSleeper(ctrl)
			// 試行間の sleep は maxAttempts-1 回。
			sleeper.EXPECT().Sleep(gomock.Any(), gomock.Any()).Return(nil).Times(maxAttempts - 1)

			m := driver.NewTransactionManager(db, dbCfg, logging.NewTestLogger(t), sleeper)
			err := m.Do(context.Background(), func(context.Context) error { return nil })

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("リトライ不可エラーは1回で返し待機しない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			db.EXPECT().Begin(gomock.Any()).Return(nil, nonRetryablePgErr).Times(1)
			sleeper := mock_clock.NewMockSleeper(ctrl)
			// リトライしないため待機は発生しない。
			sleeper.EXPECT().Sleep(gomock.Any(), gomock.Any()).Times(0)

			dbCfg := config.NewDatabaseConfig(config.MockConfigForTest(t))
			m := driver.NewTransactionManager(db, dbCfg, logging.NewTestLogger(t), sleeper)
			err := m.Do(context.Background(), func(context.Context) error { return nil })

			require.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("リトライ待機中にcontextがキャンセルされると直前のエラーを正規化して返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			// 1回目のリトライ可能エラー後、待機（Sleep）が ctx 打ち切りで失敗するため再試行されない。
			db.EXPECT().Begin(gomock.Any()).Return(nil, retryablePgErr).Times(1)
			sleeper := mock_clock.NewMockSleeper(ctrl)
			sleeper.EXPECT().Sleep(gomock.Any(), gomock.Any()).Return(context.Canceled).Times(1)

			dbCfg := config.NewDatabaseConfig(config.MockConfigForTest(t))
			m := driver.NewTransactionManager(db, dbCfg, logging.NewTestLogger(t), sleeper)
			err := m.Do(context.Background(), func(context.Context) error { return nil })

			// 待機の context.Canceled ではなく、リトライ対象だった元の失敗（40001）を
			// 正規化した ErrUnavailable が返る。
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}

func TestTxManager_Do_ContextError(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Beginがcontext.DeadlineExceededを返すとErrUnavailableへ正規化される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			// context.DeadlineExceeded はリトライ対象外のため Begin は1回のみ。
			db.EXPECT().Begin(gomock.Any()).Return(nil, context.DeadlineExceeded).Times(1)
			sleeper := mock_clock.NewMockSleeper(ctrl)
			// リトライ対象外のため待機は発生しない。
			sleeper.EXPECT().Sleep(gomock.Any(), gomock.Any()).Times(0)

			dbCfg := config.NewDatabaseConfig(config.MockConfigForTest(t))
			m := driver.NewTransactionManager(db, dbCfg, logging.NewTestLogger(t), sleeper)
			err := m.Do(context.Background(), func(context.Context) error { return nil })

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("Beginがcontext.Canceledを返すとErrCanceledへ正規化される", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			db.EXPECT().Begin(gomock.Any()).Return(nil, context.Canceled).Times(1)
			sleeper := mock_clock.NewMockSleeper(ctrl)
			// リトライ対象外のため待機は発生しない。
			sleeper.EXPECT().Sleep(gomock.Any(), gomock.Any()).Times(0)

			dbCfg := config.NewDatabaseConfig(config.MockConfigForTest(t))
			m := driver.NewTransactionManager(db, dbCfg, logging.NewTestLogger(t), sleeper)
			err := m.Do(context.Background(), func(context.Context) error { return nil })

			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}
