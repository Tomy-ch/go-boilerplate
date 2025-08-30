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

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// NewTestInstances はリポジトリのテスト用で必要なインスタンスを生成して返します。
func NewTestInstances(t *testing.T) (
	context.Context, *sql.DB, tx.Manager, *zap.Logger, *time.Location,
) {
	t.Helper()

	cfg := config.MockConfigForTest(t)
	db, err := rdbdriver.NewDB(cfg)
	require.NoError(t, err)

	nopLogger := zap.NewNop()
	txm := rdbdriver.NewTransactionManager(cfg, db)

	location, err := time.LoadLocation(cfg.OSTimeZone())
	require.NoError(t, err)

	ctx := context.Background()
	return ctx, db, txm, nopLogger, location
}
