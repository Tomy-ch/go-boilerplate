// Package testkit は RDB のテスト用インスタンスを提供します。
package testkit

import (
	"context"
	"sync"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/driver/loggingdb"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/require"
)

var rollbackForTestError = xerrors.New("rollback for test")

var (
	testDB  driver.DatabaseDriver
	initErr error
	dbOnce  sync.Once
	// txLock は、テスト用のトランザクションマネージャーでトランザクションを開始する際のロックです。
	// これにより、テストが並行して実行される場合でも、トランザクションの競合を防止します。
	// テスト数が増加し、ボトルネックとなる場合はテスト用のDocker一時コンテナを用意するライブラリの導入を検討してください。
	txLock sync.Mutex
)

type TransactionRunner interface {
	WithinTx(fn func(ctx context.Context))
}

// testTxManager はテスト用のトランザクションマネージャーを表します。
type testTxManager struct {
	inner tx.Manager
	t     *testing.T
}

// NewTestDB は、テスト用のデータベースドライバーを生成します。
func NewTestDB(t *testing.T) driver.DatabaseDriver {
	t.Helper()
	return getTestDB(t)
}

// NewTestLoggingProvider は、テスト用のログ付きDBプロバイダーを生成します。
func NewTestLoggingProvider(t *testing.T) loggingdb.DBProvider {
	t.Helper()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	obsCfg := config.NewObservabilityConfig(cfg)
	tracer := observability.NewNoopTracerFactory(t)

	mockLogger := logging.NewTestLogger(t)
	lf := logging.NewTestLogFieldBuilder(t)

	return loggingdb.NewLoggingDBProvider(getTestDB(t), dbCfg, obsCfg, mockLogger, lf, tracer)
}

// NewTestTransactionManager は、テスト用のトランザクションマネージャーを生成します。
func NewTestTransactionManager(t *testing.T) TransactionRunner {
	t.Helper()
	testLogger := logging.NewTestLogger(t)

	innerTxm := driver.NewTransactionManager(getTestDB(t), testLogger)

	txm := &testTxManager{
		inner: innerTxm,
		t:     t,
	}

	return txm
}

// WithinTx は、テスト用のトランザクションマネージャーでトランザクションを開始し、引数で渡されたfnを実行し、最後にロールバックします。
//
// 使用例:
//
//	txm.WithinTx(func(ctx context.Context) {
//	  // トランザクション内の処理
//	})
func (t *testTxManager) WithinTx(fn func(ctx context.Context)) {
	t.t.Helper()

	txLock.Lock()
	defer txLock.Unlock()

	baseCtx := context.Background()

	err := t.inner.Do(baseCtx, func(txCtx context.Context) error {
		fn(txCtx)
		return rollbackForTestError
	})

	if xerrors.Is(err, rollbackForTestError) {
		return
	}
	require.NoError(t.t, err)
}

// getTestDB は、テスト用のデータベースドライバーを生成します。シングルトンパターンで実装されており、複数回呼び出されても同じインスタンスを返します。
func getTestDB(t *testing.T) driver.DatabaseDriver {
	t.Helper()

	dbOnce.Do(func() {
		cfg := config.MockConfigForTest(t)
		dbCfg := config.NewDatabaseConfig(cfg)
		osCfg := config.NewOperatingSystemConfig(cfg)
		dbConnCfg := config.NewDBConnectionConfig(cfg)

		testDB, initErr = driver.NewDB(dbCfg, osCfg, dbConnCfg)
	})

	require.NoError(t, initErr)
	require.NotNil(t, testDB)

	return testDB
}
