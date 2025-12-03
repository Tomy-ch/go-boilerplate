// Package rdbtest は RDB のテスト用インスタンスを提供します。
package rdbtest

import (
	"context"
	"testing"
	"time"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/logging"
	"boilerplate-go/internal/observability"
	"boilerplate-go/internal/usecase/tx"
	"boilerplate-go/pkg/xerrors"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var rollbackForTestError = xerrors.New("rollback for test")

type Manager interface {
	Do(fn func(ctx context.Context) error) error
}

// testTxManager はテスト用のトランザクションマネージャーを表します。
type testTxManager struct {
	inner tx.Manager
	t     *testing.T
}

// NewTestInstancesForNew は、リポジトリのNew関数用テストインスタンスを生成します。
func NewTestInstancesForNew(t *testing.T) (
	driver.DatabaseDriver, driver.LoggingDBProvider, observability.TracerFactory,
) {
	t.Helper()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOSConfig(cfg)
	obsCfg := config.NewObservabilityConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)

	db, err := driver.NewDB(dbCfg, osCfg, dbConnCfg)
	require.NoError(t, err)

	nopLogger := zap.NewNop()
	lf := logging.NewLogFields(obsCfg)

	loggingDBProvider := driver.NewLoggingDBProvider(db, nopLogger, lf)

	noopTF := observability.NewTestTracerFactory(t)

	return db, loggingDBProvider, noopTF
}

// NewTestInstancesForImplementedInfra は、実装済みインフラ用テストインスタンスを生成します。
func NewTestInstancesForImplementedInfra(t *testing.T) (
	driver.DatabaseDriver, Manager, driver.LoggingDBProvider, *time.Location, observability.LayerTracer,
) {
	t.Helper()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOSConfig(cfg)
	obsCfg := config.NewObservabilityConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)

	db, err := driver.NewDB(dbCfg, osCfg, dbConnCfg)
	require.NoError(t, err)

	nopLogger := zap.NewNop()
	lf := logging.NewLogFields(obsCfg)
	innerTxm := driver.NewTransactionManager(cfg, db)

	loc, err := time.LoadLocation(osCfg.TimeZone())
	require.NoError(t, err)

	loggingDBProvider := driver.NewLoggingDBProvider(db, nopLogger, lf)

	noopTF := observability.NewTestTracerFactory(t)

	tracer := noopTF.Infra()

	txm := &testTxManager{
		inner: innerTxm,
		t:     t,
	}

	return db, txm, loggingDBProvider, loc, tracer
}

// Do は、テスト用のトランザクションマネージャーでトランザクションを開始し、引数で渡されたfnを実行し、最後にロールバックします。
//
// 使用例:
//
//	err := txm.Do(func(ctx context.Context) error {
//	  // トランザクション内の処理
//	  return // nil またはエラー
//	})
//	require.Error(t, err) or require.NoError(t, err)
func (t *testTxManager) Do(fn func(ctx context.Context) error) error {
	t.t.Helper()
	baseCtx := context.Background()

	err := t.inner.Do(baseCtx, func(txCtx context.Context) error {
		if err := fn(txCtx); err != nil {
			return err
		}
		return rollbackForTestError
	})

	if xerrors.Is(err, rollbackForTestError) {
		return nil
	}
	return err
}
