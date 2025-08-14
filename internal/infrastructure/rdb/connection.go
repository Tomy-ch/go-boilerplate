// Package rdb は、RDBの接続のための基盤的な機能を提供します。
package rdb

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
func NewDB(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DatabaseDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open DB: %w", err)
	}

	// 接続プール設定
	db.SetMaxOpenConns(cfg.DBMaxOpenConns())
	db.SetMaxIdleConns(cfg.DBMaxIdleConns())
	db.SetConnMaxLifetime(cfg.DBConnMaxLifetime())
	db.SetConnMaxIdleTime(cfg.DBConnMaxIdleTime())

	// 疎通確認
	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping DB: %w", err)
	}

	return db, nil
}
