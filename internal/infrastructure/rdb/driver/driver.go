package rdbdriver

import (
	"context"
	"database/sql"

	"go.uber.org/zap"
)

// ResolveDriver は、DB接続のドライバーを提供します。
func ResolveDriver(ctx context.Context, db *sql.DB) DBTX {
	tx, ok := ctx.Value(txKey{}).(*sql.Tx)
	if ok {
		return tx
	}
	return db
}

// ResolveDriverWithLog は、DB接続し、ログの出力をするドライバーを提供します。
func ResolveDriverWithLog(ctx context.Context, db *sql.DB, logger *zap.Logger) DBTX {
	return NewLoggingDB(ResolveDriver(ctx, db), logger)
}
