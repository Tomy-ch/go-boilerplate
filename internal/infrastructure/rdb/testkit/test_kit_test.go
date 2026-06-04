package testkit

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"

	"go-boilerplate/internal/infrastructure/rdb/driver"

	"github.com/stretchr/testify/require"
)

func TestNewTestDB(t *testing.T) {
	t.Parallel()
	db := NewTestDB(t)
	// 返る DB が生きている（接続可能）ことを検証する。
	require.NoError(t, db.Ping(context.Background()))
}

func TestNewTestLoggingProvider(t *testing.T) {
	t.Parallel()
	provider := NewTestLoggingProvider(t)
	// provider が必要な依存を結線して返すことを検証する。
	require.NotNil(t, provider.Logger())
	require.NotNil(t, provider.LogFields())
	require.NotNil(t, provider.DBConfig())
	require.NotNil(t, provider.ObservabilityConfig())
	require.NotNil(t, provider.LayerTracer())
	require.NotNil(t, provider.NewLoggingDB(context.Background()))
}

func TestNewTestTransactionManager(t *testing.T) {
	t.Parallel()
	runner := NewTestTransactionManager(t)
	// 公開 API 経由で WithinTx がコールバックを実行する（実トランザクションを開始しロールバックする）ことを検証する。
	ran := false
	runner.WithinTx(func(context.Context) { ran = true })
	require.True(t, ran)
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
