package driver

import (
	"context"
	"database/sql"
	"time"

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
	db       DBTX
	ctx      context.Context
	provider LoggingDBProvider
}

func (dwl *dbWithLogging) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	start := time.Now()
	res, err := dwl.db.ExecContext(ctx, query, args...)
	fields := dwl.buildSQLLogFields(ctx, "ExecContext", query, time.Since(start), args, err)
	dwl.logQueryResult("SQL Exec", fields, err)
	return res, err
}

func (dwl *dbWithLogging) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	start := time.Now()
	stmt, err := dwl.db.PrepareContext(ctx, query)
	fields := dwl.buildSQLLogFields(ctx, "PrepareContext", query, time.Since(start), nil, err)
	dwl.logQueryResult("SQL Prepare", fields, err)
	return stmt, err
}

func (dwl *dbWithLogging) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	start := time.Now()
	rows, err := dwl.db.QueryContext(ctx, query, args...)
	fields := dwl.buildSQLLogFields(ctx, "QueryContext", query, time.Since(start), args, err)
	dwl.logQueryResult("SQL Query", fields, err)
	return rows, err
}

func (dwl *dbWithLogging) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	start := time.Now()
	row := dwl.db.QueryRowContext(ctx, query, args...)
	fields := dwl.buildSQLLogFields(ctx, "QueryRowContext", query, time.Since(start), args, nil)
	dwl.logQueryResult("SQL QueryRow", fields, nil)
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
	return dwl.provider.logFields().BuildSQLFields(sqlIn)
}

// logQueryResult は、SQLクエリの実行結果をログ出力します。
func (dwl *dbWithLogging) logQueryResult(msg string, fields []*logging.Field, err error) {
	logger := dwl.provider.logger().Named("driver.Query").CallerSkip(callSkip)
	if err == nil {
		logger.Info(msg, fields...)
	} else {
		logger.Error(msg, fields...)
	}
}
