//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

// Package driver は、RDBの接続のための基盤的な機能を提供します。
package driver

import (
	"context"
	"database/sql"
	"fmt"

	"boilerplate-go/internal/config"
)

type DatabaseDriver interface {
	DBTX

	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
	PingContext(ctx context.Context) error
	Close() error
}

type dbDriver struct{ *sql.DB }

// NewDB は Postgres のDB接続を初期化して返します。
func NewDB(
	dbCfg *config.DatabaseConfig, osCfg *config.OperationSystemConfig, dbConnCfg *config.DBConnectionConfig,
) (DatabaseDriver, error) {
	db, err := sql.Open(dbCfg.Driver(), dbCfg.DSNWithTimeZone(osCfg))
	if err != nil {
		return nil, fmt.Errorf("failed to open DB: %w", err)
	}

	// 接続プール設定
	db.SetMaxOpenConns(dbConnCfg.MaxOpenConns())
	db.SetMaxIdleConns(dbConnCfg.MaxIdleConns())
	db.SetConnMaxLifetime(dbConnCfg.MaxLifetime())
	db.SetConnMaxIdleTime(dbConnCfg.MaxIdleTime())

	// 疎通確認
	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping DB: %w", err)
	}

	return &dbDriver{DB: db}, nil
}

// ExecContext は、DB.ExecContextを呼び出します。
func (d *dbDriver) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return d.DB.ExecContext(ctx, query, args...)
}

// PrepareContext は、DB.PrepareContextを呼び出します。
func (d *dbDriver) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return d.DB.PrepareContext(ctx, query)
}

// QueryContext は、DB.QueryContextを呼び出します。
func (d *dbDriver) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return d.DB.QueryContext(ctx, query, args...)
}

// QueryRowContext は、DB.QueryRowContextを呼び出します。
func (d *dbDriver) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return d.DB.QueryRowContext(ctx, query, args...)
}

// BeginTx は、DB.BeginTxを呼び出します。
func (d *dbDriver) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return d.DB.BeginTx(ctx, opts)
}

// PingContext は、DB.PingContextを呼び出します。
func (d *dbDriver) PingContext(ctx context.Context) error {
	return d.DB.PingContext(ctx)
}

// Close は、DB.Closeを呼び出します。
func (d *dbDriver) Close() error {
	return d.DB.Close()
}
