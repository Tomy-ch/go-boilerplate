// Package rdbtest は RDB のテスト用インスタンスを提供します。
package rdbtest

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"boilerplate-go/internal/config"
	rdbdriver "boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/observability"
	"boilerplate-go/internal/usecase/tx"
	"boilerplate-go/pkg/xerrors"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

type Manager interface {
	Do(fn func(ctx context.Context) error) error
}

// testTxManager はテスト用のトランザクションマネージャーを表します。
type testTxManager struct {
	inner tx.Manager
	t     *testing.T
}

var rollbackForTestError = xerrors.New("rollback for test")

// NewTestInstancesForNew は、リポジトリのNew関数用テストインスタンスを生成します。
func NewTestInstancesForNew(t *testing.T) (
	*sql.DB, *zap.Logger, observability.TracerFactory,
) {
	t.Helper()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOSConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)

	db, err := rdbdriver.NewDB(dbCfg, osCfg, dbConnCfg)
	require.NoError(t, err)

	nopLogger := zap.NewNop()
	noopTP := noop.NewTracerProvider()
	noopTF := observability.NewTracerFactory(noopTP, nopLogger)

	return db, nopLogger, noopTF
}

// NewTestInstancesForImplementedInfra は、実装済みインフラ用テストインスタンスを生成します。
func NewTestInstancesForImplementedInfra(t *testing.T) (
	*sql.DB, Manager, *zap.Logger, *time.Location, observability.LayerTracer,
) {
	t.Helper()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOSConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)

	db, err := rdbdriver.NewDB(dbCfg, osCfg, dbConnCfg)
	require.NoError(t, err)

	nopLogger := zap.NewNop()
	innerTxm := rdbdriver.NewTransactionManager(cfg, db)

	loc, err := time.LoadLocation(osCfg.OSTimeZone())
	require.NoError(t, err)

	noopTP := noop.NewTracerProvider()
	noopTF := observability.NewTracerFactory(noopTP, nopLogger)

	tracer := noopTF.Infra()

	txm := &testTxManager{
		inner: innerTxm,
		t:     t,
	}

	return db, txm, nopLogger, loc, tracer
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
