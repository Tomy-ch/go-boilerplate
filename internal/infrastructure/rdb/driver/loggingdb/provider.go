//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

// Package loggingdb は、ログ付きのDB接続を提供します。
package loggingdb

import (
	"context"

	"boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/logging"
)

type DBProvider interface {
	NewLoggingDB(ctx context.Context) driver.DBTX
	Logger() logging.Logger
	LogFields() logging.LogFieldBuilder
}

// provider は、ログ付きDB接続を提供します。
type provider struct {
	db driver.DatabaseDriver
	l  logging.Logger
	lf logging.LogFieldBuilder
}

// NewLoggingDBProvider は、DBTXProviderの新しいインスタンスを生成します。
func NewLoggingDBProvider(db driver.DatabaseDriver, log logging.Logger, lf logging.LogFieldBuilder) DBProvider {
	return &provider{
		db: db,
		l:  log,
		lf: lf,
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
