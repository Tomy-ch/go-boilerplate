//go:generate mockgen -source=$GOFILE -destination=mock/mock_provider.gen.go -package=mock_$GOPACKAGE

// Package loggingdb は、ログ付きのDB接続を提供します。
package loggingdb

import (
	"context"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
)

// DBProvider は、コンテキストを受け取り DBTX を生成するファクトリインタフェースです。
type DBProvider interface {
	NewLoggingDB(ctx context.Context) driver.DBTX
}

// provider は、ログ付きDB接続を提供します。
type provider struct {
	db     driver.DatabaseDriver
	dbCfg  *config.DatabaseConfig
	obsCfg *config.ObservabilityConfig
	l      logging.Logger
	lf     logging.LogFieldBuilder
	tracer observability.LayerTracer
}

// NewLoggingDBProvider は、DBProviderの新しいインスタンスを生成します。
func NewLoggingDBProvider(
	db driver.DatabaseDriver,
	dbCfg *config.DatabaseConfig,
	obsCfg *config.ObservabilityConfig,
	log logging.Logger,
	lf logging.LogFieldBuilder,
	tracer observability.TracerFactory,
) DBProvider {
	return &provider{
		db:     db,
		dbCfg:  dbCfg,
		obsCfg: obsCfg,
		l:      log,
		lf:     lf,
		tracer: tracer.Infra(),
	}
}

// NewLoggingDB は、DBTXを生成します。
func (p *provider) NewLoggingDB(ctx context.Context) driver.DBTX {
	return &dbWithLogging{
		db:       driver.New(ctx, p.db),
		provider: p,
	}
}
