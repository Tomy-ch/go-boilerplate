package testkit

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/system"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestDB(t *testing.T) {
	t.Parallel()
	db := NewTestDB(t)
	// 返る DB が生きている（接続可能）ことを検証する。
	require.NoError(t, db.Ping(context.Background()))
}

func TestNewTestTransactionRunner(t *testing.T) {
	t.Parallel()
	runner := NewTestTransactionRunner(t)
	// 公開 API 経由で WithinTx がコールバックを実行する（実トランザクションを開始しロールバックする）ことを検証する。
	ran := false
	runner.WithinTx(func(context.Context) { ran = true })
	assert.True(t, ran)
}

func Test_testTxRunner_Do(t *testing.T) {
	t.Parallel()
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOperatingSystemConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)

	testLogger := logging.NewTestLogger(t)

	db, err := driver.NewDB(dbCfg, osCfg, dbConnCfg)
	require.NoError(t, err)
	innerTxm := driver.NewTransactionManager(db, dbCfg, testLogger, system.NewSleeper())

	t.Run("実行時にエラーが発生しない場合、正常に終了すること", func(t *testing.T) {
		t.Parallel()
		txm := &testTxRunner{
			inner: innerTxm,
			t:     t,
		}
		txm.WithinTx(func(_ context.Context) {})
	})
}
