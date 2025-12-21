package rdbtest

import (
	"context"
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/infrastructure/rdb/driver"

	"github.com/stretchr/testify/require"
)

func TestNewTestDBWithLoggingProvider(t *testing.T) {
	t.Parallel()
	db, provider := NewTestDBWithLoggingProvider(t)
	require.NotNil(t, db)
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
	osCfg := config.NewOSConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)

	db, err := driver.NewDB(dbCfg, osCfg, dbConnCfg)
	require.NoError(t, err)
	innerTxm := driver.NewTransactionManager(cfg, db)

	t.Run("実行時にエラーが発生しない場合、正常に終了すること", func(t *testing.T) {
		t.Parallel()
		txm := &testTxManager{
			inner: innerTxm,
			t:     t,
		}
		txm.WithinTx(func(_ context.Context) {})
	})
}
