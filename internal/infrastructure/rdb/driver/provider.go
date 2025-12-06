//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE
package driver

import (
	"context"

	"boilerplate-go/internal/logging"
)

type LoggingDBProvider interface {
	NewLoggingDB(ctx context.Context) DBTX
	logger() logging.Logger
	logFields() logging.LogFieldBuilder
}

// loggingDBProvider は、DBTXを提供します。
type loggingDBProvider struct {
	db DatabaseDriver
	l  logging.Logger
	lf logging.LogFieldBuilder
}

// NewLoggingDBProvider は、DBTXProviderの新しいインスタンスを生成します。
func NewLoggingDBProvider(db DatabaseDriver, log logging.Logger, lf logging.LogFieldBuilder) LoggingDBProvider {
	return &loggingDBProvider{
		db: db,
		l:  log,
		lf: lf,
	}
}

// NewLoggingDB は、DBTXを生成します。
func (cd *loggingDBProvider) NewLoggingDB(ctx context.Context) DBTX {
	return &dbWithLogging{
		db:       New(ctx, cd.db),
		ctx:      ctx,
		provider: cd,
	}
}

// logger は、ロガーを返します。
func (cd *loggingDBProvider) logger() logging.Logger {
	return cd.l
}

// logFields は、ログフィールドのビルダーを返します。
func (cd *loggingDBProvider) logFields() logging.LogFieldBuilder {
	return cd.lf
}
