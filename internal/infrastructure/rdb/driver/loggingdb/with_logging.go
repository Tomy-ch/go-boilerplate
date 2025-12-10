package loggingdb

import (
	"context"
	"database/sql"
	"time"

	"boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/logging"
	"boilerplate-go/internal/observability"
)

const (
	callSkip = 3
	layer    = "infrastructure"
	pkg      = "driver"
)

// dbWithLogging は DBTX をラップしてログを出してから実処理へ委譲する。
type dbWithLogging struct {
	db       driver.DBTX
	ctx      context.Context
	provider DBProvider
}

func (dwl *dbWithLogging) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	start := time.Now()
	res, err := dwl.db.ExecContext(ctx, query, args...)
	duration := time.Since(start)
	fields := dwl.buildSQLLogFields(ctx, "ExecContext", query, duration, args, err)
	dwl.logQueryResult("SQL Exec", duration, fields, err)
	return res, err
}

func (dwl *dbWithLogging) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	start := time.Now()
	stmt, err := dwl.db.PrepareContext(ctx, query)
	duration := time.Since(start)
	fields := dwl.buildSQLLogFields(ctx, "PrepareContext", query, duration, nil, err)
	dwl.logQueryResult("SQL Prepare", duration, fields, err)
	return stmt, err
}

func (dwl *dbWithLogging) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	start := time.Now()
	rows, err := dwl.db.QueryContext(ctx, query, args...)
	duration := time.Since(start)
	fields := dwl.buildSQLLogFields(ctx, "QueryContext", query, duration, args, err)
	dwl.logQueryResult("SQL Query", duration, fields, err)
	return rows, err
}

func (dwl *dbWithLogging) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	start := time.Now()
	row := dwl.db.QueryRowContext(ctx, query, args...)
	duration := time.Since(start)
	fields := dwl.buildSQLLogFields(ctx, "QueryRowContext", query, duration, args, nil)
	dwl.logQueryResult("SQL QueryRow", duration, fields, nil)
	return row
}

// buildSQLLogFields は、SQLクエリのログ出力用フィールドを構築します。
func (dwl *dbWithLogging) buildSQLLogFields(
	ctx context.Context, funcName, query string, duration time.Duration, args []any, err error,
) []*logging.Field {
	traceCtx := observability.ExtractSpan(ctx)
	sqlIn := logging.SQLFieldsInput{
		Layer:    layer,
		PkgName:  pkg,
		FuncName: funcName,
		SpanName: observability.BuildSpanName(layer, pkg, funcName),

		Query:    query,
		Duration: duration,
		Args:     args,
		Err:      err,
		TraceID:  traceCtx.TraceID(),
		SpanID:   traceCtx.SpanID(),
	}
	return dwl.provider.LogFields().BuildSQLFields(sqlIn)
}

// logQueryResult は、SQLクエリの実行結果をログ出力します。
func (dwl *dbWithLogging) logQueryResult(
	msg string, duration time.Duration, fields []*logging.Field, err error,
) {
	logger := dwl.provider.Logger().Named("driver.Query").CallerSkip(callSkip)
	threshold := dwl.provider.DBConfig().SlowQueryWarnThreshold()
	switch {
	case err != nil:
		logger.Error(msg, fields...)
	case threshold > 0 && duration > threshold:
		logger.Warn(msg, fields...)
	default:
		logger.Info(msg, fields...)
	}
}
