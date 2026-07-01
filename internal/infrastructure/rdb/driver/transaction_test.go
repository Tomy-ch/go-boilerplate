package driver

import (
	"testing"

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
