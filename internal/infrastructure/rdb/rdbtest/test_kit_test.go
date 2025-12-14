package rdbtest

import (
	"context"
	"testing"
	"time"

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

func TestNewNoopTracerFactory(t *testing.T) {
	t.Parallel()
	tf := NewNoopTracerFactory(t)
	require.NotNil(t, tf)
}

func TestNewNoopInfraLayerTracer(t *testing.T) {
	t.Parallel()
	lt := NewNoopInfraLayerTracer(t)
	require.NotNil(t, lt)
}

func TestNewTestLocation(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	osCfg := config.NewOSConfig(cfg)

	expected, err := time.LoadLocation(osCfg.TimeZone())
	require.NoError(t, err)

	actual := NewTestLocation(t)
	require.Equal(t, expected, actual)
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
