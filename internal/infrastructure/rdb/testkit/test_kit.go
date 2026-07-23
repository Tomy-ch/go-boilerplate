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

// testTxAdvisoryLockKey は、テスト tx を共有 DB 内で全プロセス横断に直列化するための advisory lock キー。
// go test は各パッケージを別プロセスで並列実行するため、プロセス内 txLock だけでは CASCADE TRUNCATE
// 同士が FK 関連テーブルのロックを交差させて deadlock しうる。advisory lock（DB 単位）で防ぐ。
const testTxAdvisoryLockKey = 8_246_913

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
	db    driver.DatabaseDriver
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

	db := getTestDB(t)
	dbCfg := config.NewDatabaseConfig(config.MockConfigForTest(t))
	innerTxm := driver.NewTransactionManager(db, dbCfg, testLogger, system.NewSleeper())

	runner := &testTxRunner{
		inner: innerTxm,
		db:    db,
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
		// 共有 DB を並列パッケージプロセス間で直列化する（tx 終了で自動解放）。プロセス内 txLock は
		// 別プロセスの CASCADE TRUNCATE と競合するロック交差を防げないため advisory lock で補う。
		if _, lockErr := driver.New(txCtx, t.db).
			Exec(txCtx, "SELECT pg_advisory_xact_lock($1)", testTxAdvisoryLockKey); lockErr != nil {
			return lockErr
		}
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
