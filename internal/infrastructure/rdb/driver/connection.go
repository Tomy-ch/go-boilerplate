// Package driver は、RDBの接続のための基盤的な機能を提供します。
package driver

import (
	"context"
	"database/sql"

	// pgx driver for db connection (required for runtime registration)
	_ "github.com/jackc/pgx/v5/stdlib"
)

// txKey は、トランザクションマネージャを識別するためのコンテキストキーです。
type txKey struct{}

// DBTX は RDBMSが期待する最小インターフェイス
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// New は、DB接続のドライバーを提供します。
func New(ctx context.Context, db DatabaseDriver) DBTX {
	tx, ok := ctx.Value(txKey{}).(*sql.Tx)
	if ok {
		return tx
	}
	return db
}
