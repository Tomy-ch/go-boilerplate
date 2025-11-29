// Package rdbdriver は、RDBの接続のための基盤的な機能を提供します。
package rdbdriver

import (
	"context"
	"database/sql"
	"fmt"

	"boilerplate-go/internal/config"

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

// NewDB は Postgres のDB接続を初期化して返します。
func NewDB(
	dbCfg *config.DatabaseConfig, osCfg *config.OperationSystemConfig, dbConnCfg *config.DBConnectionConfig,
) (*sql.DB, error) {
	db, err := sql.Open(dbCfg.DatabaseDriver(), dbCfg.DatabaseDSN(osCfg))
	if err != nil {
		return nil, fmt.Errorf("failed to open DB: %w", err)
	}

	// 接続プール設定
	db.SetMaxOpenConns(dbConnCfg.DBMaxOpenConns())
	db.SetMaxIdleConns(dbConnCfg.DBMaxIdleConns())
	db.SetConnMaxLifetime(dbConnCfg.DBConnMaxLifetime())
	db.SetConnMaxIdleTime(dbConnCfg.DBConnMaxIdleTime())

	// 疎通確認
	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping DB: %w", err)
	}

	return db, nil
}
