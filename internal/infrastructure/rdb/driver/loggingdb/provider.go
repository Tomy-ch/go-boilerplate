//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

// Package loggingdb は、ログ付きのDB接続を提供します。
package loggingdb

import (
	"context"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/logging"
	"boilerplate-go/internal/observability"
)

type DBProvider interface {
	NewLoggingDB(ctx context.Context) driver.DBTX
	Logger() logging.Logger
	LogFields() logging.LogFieldBuilder
	DBConfig() *config.DatabaseConfig
	ObservabilityConfig() *config.ObservabilityConfig
	LayerTracer() observability.LayerTracer
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

// NewLoggingDBProvider は、DBTXProviderの新しいインスタンスを生成します。
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
		ctx:      ctx,
		provider: p,
	}
}

// Logger は、ロガーを返します。
func (p *provider) Logger() logging.Logger {
	return p.l
}

// LogFields は、ログフィールドのビルダーを返します。
func (p *provider) LogFields() logging.LogFieldBuilder {
	return p.lf
}

// DBConfig は、データベース設定を返します。
func (p *provider) DBConfig() *config.DatabaseConfig {
	return p.dbCfg
}

// ObservabilityConfig は、観測可能性設定を返します。
func (p *provider) ObservabilityConfig() *config.ObservabilityConfig {
	return p.obsCfg
}

// LayerTracer は、レイヤートレーサーを返します。
func (p *provider) LayerTracer() observability.LayerTracer {
	return p.tracer
}
