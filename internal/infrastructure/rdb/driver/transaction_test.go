package driver

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/system"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"
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

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定値からリトライ上限とbackoffが構成される", func(t *testing.T) {
			t.Parallel()

			manager := NewTransactionManager(db, dbCfg, testLogger, system.NewSleeper())
			require.NotNil(t, manager)

			txm, ok := manager.(*txManager)
			require.True(t, ok)
			assert.Equal(t, dbCfg.TxMaxRetries(), txm.maxAttempts)
			assert.Equal(t, dbCfg.TxRetryBaseBackoff(), txm.backoff.Initial)
			assert.Equal(t, dbCfg.TxRetryMaxBackoff(), txm.backoff.Max)
		})

		t.Run("設定値が0以下の場合は既定値へフォールバックする", func(t *testing.T) {
			t.Parallel()

			// ゼロ値の DatabaseConfig は各リトライ設定が 0 のため、既定値へフォールバックする。
			manager := NewTransactionManager(db, &config.DatabaseConfig{}, testLogger, system.NewSleeper())
			require.NotNil(t, manager)

			txm, ok := manager.(*txManager)
			require.True(t, ok)
			assert.Equal(t, defaultTxMaxAttempts, txm.maxAttempts)
			assert.Equal(t, defaultTxBackoffInitial, txm.backoff.Initial)
			assert.Equal(t, defaultTxBackoffMax, txm.backoff.Max)
		})
	})
}

func Test_normalizeTxResult(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilはnilを返す", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, normalizeTxResult(nil))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("PgErrorはapperrorへ正規化される", func(t *testing.T) {
			t.Parallel()
			got := normalizeTxResult(&pgconn.PgError{Code: "23505"})
			require.ErrorIs(t, got, apperror.ErrConflict)
		})

		t.Run("接続不可エラー(08xxx)はapperrorへ正規化される", func(t *testing.T) {
			t.Parallel()
			got := normalizeTxResult(&pgconn.PgError{Code: "08006"})
			require.ErrorIs(t, got, apperror.ErrUnavailable)
		})

		t.Run("context.Canceledはapperrorへ正規化される", func(t *testing.T) {
			t.Parallel()
			got := normalizeTxResult(context.Canceled)
			require.ErrorIs(t, got, apperror.ErrCanceled)
		})

		t.Run("context.DeadlineExceededはapperrorへ正規化される", func(t *testing.T) {
			t.Parallel()
			got := normalizeTxResult(context.DeadlineExceeded)
			require.ErrorIs(t, got, apperror.ErrUnavailable)
		})

		t.Run("fnが返した非DBエラーは正規化せず素通しする", func(t *testing.T) {
			t.Parallel()
			appErr := xerrors.Wrap(apperror.ErrValidation, "boom")
			got := normalizeTxResult(appErr)
			require.ErrorIs(t, got, apperror.ErrValidation)
		})
	})
}

func Test_txManager_doOnce(t *testing.T) {
	t.Parallel()
	t.Skip("Test_txManager_Do（driver_test パッケージ）の実 DB / mock テストでカバー")
}

func Test_txManager_rollback(t *testing.T) {
	t.Parallel()
	t.Skip("Test_txManager_Do の「rollback失敗時はエラーログを出力し元のエラーを返す」ケースでカバー")
}
