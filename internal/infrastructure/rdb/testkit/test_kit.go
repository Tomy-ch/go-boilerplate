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

var errRollbackForTest = xerrors.New("rollback for test")

var (
	testDB  driver.DatabaseDriver
	errInit error
	dbOnce  sync.Once
	// txLock は、テスト用のトランザクションマネージャーでトランザクションを開始する際のロックです。
	// これにより、テストが並行して実行される場合でも、トランザクションの競合を防止します。
	// テスト数が増加し、ボトルネックとなる場合はテスト用のDocker一時コンテナを用意するライブラリの導入を検討してください。
	txLock sync.Mutex
)

// TransactionRunner は、テスト用に関数をトランザクション内で実行するインターフェースです。
type TransactionRunner interface {
	WithinTx(fn func(ctx context.Context))
}

// testTxRunner はテスト用のトランザクションランナーを表します。
type testTxRunner struct {
	inner tx.Manager
	t     *testing.T
}

// NewTestDB は、テスト用の共有データベースドライバー（シングルトン）を取得します。
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

// NewTestTransactionRunner は、テスト用のトランザクションランナーを生成します。
func NewTestTransactionRunner(t *testing.T) TransactionRunner {
	t.Helper()
	testLogger := logging.NewTestLogger(t)

	innerTxm := driver.NewTransactionManager(getTestDB(t), testLogger)

	runner := &testTxRunner{
		inner: innerTxm,
		t:     t,
	}

	return runner
}

// WithinTx は、テスト用のトランザクションマネージャーでトランザクションを開始し、引数で渡されたfnを実行し、最後にロールバックします。
//
// 使用例:
//
//	txm.WithinTx(func(ctx context.Context) {
//	  // トランザクション内の処理
//	})
func (t *testTxRunner) WithinTx(fn func(ctx context.Context)) {
	t.t.Helper()

	txLock.Lock()
	defer txLock.Unlock()

	baseCtx := context.Background()

	err := t.inner.Do(baseCtx, func(txCtx context.Context) error {
		fn(txCtx)
		return errRollbackForTest
	})

	if xerrors.Is(err, errRollbackForTest) {
		return
	}
	require.NoError(t.t, err)
}

// getTestDB は、テスト用の共有データベースドライバーを返します（初回呼び出し時のみ生成）。
func getTestDB(t *testing.T) driver.DatabaseDriver {
	t.Helper()

	dbOnce.Do(func() {
		cfg := config.MockConfigForTest(t)
		dbCfg := config.NewDatabaseConfig(cfg)
		osCfg := config.NewOperatingSystemConfig(cfg)
		dbConnCfg := config.NewDBConnectionConfig(cfg)

		testDB, errInit = driver.NewDB(dbCfg, osCfg, dbConnCfg)
	})

	require.NoError(t, errInit)
	require.NotNil(t, testDB)

	return testDB
}
