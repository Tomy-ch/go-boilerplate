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

	sqlExec        = "Execute statement"
	sqlPrepare     = "Prepare statement"
	sqlQuery       = "Query multiple rows"
	sqlQuerySingle = "Query single row"

	execContext     = "ExecContext"
	prepareContext  = "PrepareContext"
	queryContext    = "QueryContext"
	queryRowContext = "QueryRowContext"
)

// dbWithLogging は DBTX をラップしてログを出してから実処理へ委譲する。
type dbWithLogging struct {
	db       driver.DBTX
	ctx      context.Context
	provider DBProvider
}

func (dwl *dbWithLogging) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	start := time.Now()

	tc, _, end := observability.StartSpanWithParent(ctx, dwl.provider.LayerTracer(), observability.BuildSpanName(layer, pkg, execContext))
	defer end()

	fields := dwl.buildSQLStartLogFields(tc, execContext)
	dwl.provider.Logger().Named(layer).CallerSkip(callSkip).Info(sqlExec, fields...)

	res, err := dwl.db.ExecContext(ctx, query, args...)
	duration := time.Since(start)

	fields = dwl.buildSQLEndLogFields(tc, execContext, query, duration, args, err)
	dwl.logQueryResult(sqlExec, duration, fields, err)
	return res, err
}

func (dwl *dbWithLogging) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	start := time.Now()

	tc, _, end := observability.StartSpanWithParent(ctx, dwl.provider.LayerTracer(), observability.BuildSpanName(layer, pkg, prepareContext))
	defer end()

	fields := dwl.buildSQLStartLogFields(tc, prepareContext)
	dwl.provider.Logger().Named(layer).CallerSkip(callSkip).Info(sqlPrepare, fields...)

	//nolint:sqlclosecheck // wrapper: caller is responsible for closing returned *sql.Stmt
	stmt, err := dwl.db.PrepareContext(ctx, query)
	duration := time.Since(start)

	fields = dwl.buildSQLEndLogFields(tc, prepareContext, query, duration, nil, err)
	dwl.logQueryResult(sqlPrepare, duration, fields, err)
	return stmt, err
}

func (dwl *dbWithLogging) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	start := time.Now()

	tc, _, end := observability.StartSpanWithParent(ctx, dwl.provider.LayerTracer(), observability.BuildSpanName(layer, pkg, queryContext))
	defer end()

	fields := dwl.buildSQLStartLogFields(tc, queryContext)
	dwl.provider.Logger().Named(layer).CallerSkip(callSkip).Info(sqlQuery, fields...)

	//nolint:sqlclosecheck // wrapper: caller is responsible for closing returned *sql.Rows
	rows, err := dwl.db.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	fields = dwl.buildSQLEndLogFields(tc, queryContext, query, duration, args, err)
	dwl.logQueryResult(sqlQuery, duration, fields, err)
	return rows, err
}

func (dwl *dbWithLogging) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	start := time.Now()

	tc, _, end := observability.StartSpanWithParent(ctx, dwl.provider.LayerTracer(), observability.BuildSpanName(layer, pkg, queryRowContext))
	defer end()

	fields := dwl.buildSQLStartLogFields(tc, queryRowContext)
	dwl.provider.Logger().Named(layer).CallerSkip(callSkip).Info(sqlQuerySingle, fields...)

	row := dwl.db.QueryRowContext(ctx, query, args...)
	duration := time.Since(start)

	fields = dwl.buildSQLEndLogFields(tc, queryRowContext, query, duration, args, nil)
	dwl.logQueryResult(sqlQuerySingle, duration, fields, nil)
	return row
}

// buildSQLStartLogFields は、SQLクエリの開始ログ出力用フィールドを構築します。
func (dwl *dbWithLogging) buildSQLStartLogFields(tc observability.TraceContext, funcName string) []*logging.Field {
	sqlIn := logging.SQLFieldsStartInput{
		Layer:    layer,
		PkgName:  pkg,
		FuncName: funcName,
		SpanName: observability.BuildSpanName(layer, pkg, funcName),

		EventAt: time.Now(),

		TraceID:      tc.TraceID(),
		SpanID:       tc.SpanID(),
		ParentSpanID: tc.ParentSpanID(),
	}
	return dwl.provider.LogFields().BuildSQLStartFields(sqlIn)
}

// buildSQLEndLogFields は、SQLクエリの終了ログ出力用フィールドを構築します。
func (dwl *dbWithLogging) buildSQLEndLogFields(
	tc observability.TraceContext, funcName, query string, duration time.Duration, args []any, err error,
) []*logging.Field {
	sqlIn := logging.SQLFieldsEndInput{
		Layer:    layer,
		PkgName:  pkg,
		FuncName: funcName,
		SpanName: observability.BuildSpanName(layer, pkg, funcName),

		EventAt: time.Now(),

		Query:        query,
		Latency:      duration,
		Args:         args,
		Err:          err,
		TraceID:      tc.TraceID(),
		SpanID:       tc.SpanID(),
		ParentSpanID: tc.ParentSpanID(),
	}
	return dwl.provider.LogFields().BuildSQLEndFields(sqlIn)
}

// logQueryResult は、SQLクエリの実行結果をログ出力します。
func (dwl *dbWithLogging) logQueryResult(
	msg string, duration time.Duration, fields []*logging.Field, err error,
) {
	logger := dwl.provider.Logger().Named(layer).CallerSkip(callSkip)
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
