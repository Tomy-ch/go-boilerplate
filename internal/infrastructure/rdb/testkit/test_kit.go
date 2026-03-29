// Package testkit は RDB のテスト用インスタンスを提供します。
package testkit

import (
	"context"
	"sync"
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/infrastructure/rdb/driver/loggingdb"
	"boilerplate-go/internal/logging"
	"boilerplate-go/internal/observability"
	"boilerplate-go/internal/usecase/boundary/tx"
	"boilerplate-go/pkg/xerrors"

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
	cfg := config.MockConfigForTest(t)
	testLogger := logging.NewTestLogger(t)

	innerTxm := driver.NewTransactionManager(cfg, getTestDB(t), testLogger)

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
		osCfg := config.NewOperationSystemConfig(cfg)
		dbConnCfg := config.NewDBConnectionConfig(cfg)

		testDB, initErr = driver.NewDB(dbCfg, osCfg, dbConnCfg)
	})

	require.NoError(t, initErr)
	require.NotNil(t, testDB)

	return testDB
}
