//go:generate mockgen -source=$GOFILE -destination=mock/mock_driver.gen.go -package=mock_$GOPACKAGE

// Package driver は、RDBの接続のための基盤的な機能を提供します。
package driver

import (
	"context"
	"fmt"

	"go-boilerplate/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DatabaseDriver は、PostgreSQL 接続プールに対するトランザクション開始・疎通確認・統計取得を含む
// DB アクセスの最上位インタフェースです。
type DatabaseDriver interface {
	DBTX

	Begin(ctx context.Context) (pgx.Tx, error)
	Ping(ctx context.Context) error
	Close() error
	Stats() *pgxpool.Stat
}

// dbDriver は pgxpool.Pool への薄いアダプタです。
type dbDriver struct{ pool *pgxpool.Pool }

// NewDB は Postgres のDB接続を初期化して返します（クエリトレーサーなし）。
func NewDB(
	dbCfg *config.DatabaseConfig, osCfg *config.OperatingSystemConfig, dbConnCfg *config.DBConnectionConfig,
) (DatabaseDriver, error) {
	return newDB(dbCfg, osCfg, dbConnCfg, nil)
}

// NewTracedDB は、pgx クエリトレーサーを結線した Postgres のDB接続を初期化して返します。
func NewTracedDB(
	dbCfg *config.DatabaseConfig,
	osCfg *config.OperatingSystemConfig,
	dbConnCfg *config.DBConnectionConfig,
	tracer pgx.QueryTracer,
) (DatabaseDriver, error) {
	return newDB(dbCfg, osCfg, dbConnCfg, tracer)
}

// newDB は、DB接続プールを初期化する共通処理です。tracer が非 nil の場合のみクエリ計装を結線します。
func newDB(
	dbCfg *config.DatabaseConfig,
	osCfg *config.OperatingSystemConfig,
	dbConnCfg *config.DBConnectionConfig,
	tracer pgx.QueryTracer,
) (DatabaseDriver, error) {
	poolCfg, err := pgxpool.ParseConfig(DSNWithTimeZoneString(dbCfg, osCfg))
	if err != nil {
		return nil, fmt.Errorf("failed to parse DB config: %w", err)
	}

	// 接続プール設定
	poolCfg.MaxConns = dbConnCfg.MaxConns()
	poolCfg.MinConns = dbConnCfg.MinConns()
	poolCfg.MaxConnLifetime = dbConnCfg.MaxLifetime()
	poolCfg.MaxConnIdleTime = dbConnCfg.MaxIdleTime()

	if tracer != nil {
		poolCfg.ConnConfig.Tracer = tracer
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbCfg.PingTimeout())
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create DB connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping DB: %w", err)
	}

	return &dbDriver{pool: pool}, nil
}

func (d *dbDriver) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return d.pool.Exec(ctx, sql, args...)
}

func (d *dbDriver) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return d.pool.Query(ctx, sql, args...)
}

func (d *dbDriver) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return d.pool.QueryRow(ctx, sql, args...)
}

func (d *dbDriver) Begin(ctx context.Context) (pgx.Tx, error) {
	return d.pool.Begin(ctx)
}

func (d *dbDriver) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

// Close は pool を閉じます。
func (d *dbDriver) Close() error {
	d.pool.Close()
	return nil
}

func (d *dbDriver) Stats() *pgxpool.Stat {
	return d.pool.Stat()
}
