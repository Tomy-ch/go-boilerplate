// Package testkit は RDB のテスト用インスタンスを提供します。
package testkit

import (
	"context"
	"sync"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/system"
	"go-boilerplate/internal/logging"
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

// NewTestTransactionRunner は、テスト用のトランザクションランナーを生成します。
func NewTestTransactionRunner(t *testing.T) TransactionRunner {
	t.Helper()
	testLogger := logging.NewTestLogger(t)

	dbCfg := config.NewDatabaseConfig(config.MockConfigForTest(t))
	innerTxm := driver.NewTransactionManager(getTestDB(t), dbCfg, testLogger, system.NewSleeper())

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
