// Package testkit は RDB のテスト用インスタンスを提供します。
package testkit

import (
	"context"
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/infrastructure/rdb/driver/loggingdb"
	"boilerplate-go/internal/logging"
	"boilerplate-go/internal/observability"
	"boilerplate-go/internal/usecase/tx"
	"boilerplate-go/pkg/xerrors"

	"github.com/stretchr/testify/require"
)

var rollbackForTestError = xerrors.New("rollback for test")

type TransactionRunner interface {
	WithinTx(fn func(ctx context.Context))
}

// testTxManager はテスト用のトランザクションマネージャーを表します。
type testTxManager struct {
	inner tx.Manager
	t     *testing.T
}

// NewTestDBWithLoggingProvider は、テスト用のデータベースドライバーとログ付きDBプロバイダーを生成します。
func NewTestDBWithLoggingProvider(t *testing.T) (driver.DatabaseDriver, loggingdb.DBProvider) {
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)
	tracer := observability.NewNoopTracerFactory(t)

	db, err := driver.NewDB(dbCfg, osCfg, dbConnCfg)
	require.NoError(t, err)

	mockLogger := logging.NewTestLogger(t)
	lf := logging.NewTestLogFieldBuilder(t)

	loggingDBProvider := loggingdb.NewLoggingDBProvider(db, dbCfg, mockLogger, lf, tracer)

	return db, loggingDBProvider
}

// NewTestTransactionManager は、テスト用のトランザクションマネージャーを生成します。
func NewTestTransactionManager(t *testing.T) TransactionRunner {
	t.Helper()
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)

	db, err := driver.NewDB(dbCfg, osCfg, dbConnCfg)
	require.NoError(t, err)
	innerTxm := driver.NewTransactionManager(cfg, db)

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
