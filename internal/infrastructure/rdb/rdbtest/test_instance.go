// Package rdbtest は RDB のテスト用インスタンスを提供します。
package rdbtest

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"boilerplate-go/internal/config"
	rdbdriver "boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/usecase/tx"
	"boilerplate-go/pkg/xerrors"

	"github.com/stretchr/testify/require"
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

var rollbackForTestError = xerrors.New("the test was successful, so we rolled it back")

// NewTestInstances はリポジトリのテスト用で必要なインスタンスを生成して返します。
func NewTestInstances(t *testing.T) (
	*sql.DB, Manager, *zap.Logger, *time.Location,
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

	location, err := time.LoadLocation(osCfg.OSTimeZone())
	require.NoError(t, err)

	txm := &testTxManager{
		inner: innerTxm,
		t:     t,
	}

	return db, txm, nopLogger, location
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
