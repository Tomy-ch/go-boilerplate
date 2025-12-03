package rdbtest

import (
	"context"
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/pkg/xerrors"

	"github.com/stretchr/testify/require"
)

func TestNewTestInstancesForNew(t *testing.T) {
	t.Parallel()
	db, logger, tp := NewTestInstancesForNew(t)
	require.NotNil(t, db)
	require.NotNil(t, logger)
	require.NotNil(t, tp)
}

func TestNewTestInstancesForImplementedInfra(t *testing.T) {
	t.Parallel()
	db, txm, logger, location, tracer := NewTestInstancesForImplementedInfra(t)
	require.NotNil(t, db)
	require.NotNil(t, txm)
	require.NotNil(t, logger)
	require.NotNil(t, location)
	require.NotNil(t, tracer)
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

	t.Run("実行時にエラーが発生しない場合、RollbackForTestErrorが返ること", func(t *testing.T) {
		t.Parallel()
		txm := &testTxManager{
			inner: innerTxm,
			t:     t,
		}
		err := txm.Do(func(_ context.Context) error {
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("実行時にエラーが発生した場合、そのエラーが発生すること", func(t *testing.T) {
		t.Parallel()
		txm := &testTxManager{
			inner: innerTxm,
			t:     t,
		}
		testErr := xerrors.New("test error")
		err := txm.Do(func(_ context.Context) error {
			return testErr
		})
		require.ErrorIs(t, err, testErr)
	})
}
