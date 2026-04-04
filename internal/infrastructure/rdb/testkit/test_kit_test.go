package testkit

import (
	"context"
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/logging"

	"boilerplate-go/internal/infrastructure/rdb/driver"

	"github.com/stretchr/testify/require"
)

func TestNewTestDB(t *testing.T) {
	t.Parallel()
	db := NewTestDB(t)
	require.NotNil(t, db)
}

func TestNewTestLoggingProvider(t *testing.T) {
	t.Parallel()
	provider := NewTestLoggingProvider(t)
	require.NotNil(t, provider)
}

func TestNewTestTransactionManager(t *testing.T) {
	t.Parallel()
	actual := NewTestTransactionManager(t)
	require.NotNil(t, actual)
}

func Test_testTxManager_Do(t *testing.T) {
	t.Parallel()
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)

	testLogger := logging.NewTestLogger(t)

	db, err := driver.NewDB(dbCfg, osCfg, dbConnCfg)
	require.NoError(t, err)
	innerTxm := driver.NewTransactionManager(cfg, db, testLogger)

	t.Run("実行時にエラーが発生しない場合、正常に終了すること", func(t *testing.T) {
		t.Parallel()
		txm := &testTxManager{
			inner: innerTxm,
			t:     t,
		}
		txm.WithinTx(func(_ context.Context) {})
	})
}
