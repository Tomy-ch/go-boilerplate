package loggingdb

import (
	"context"
	"time"

	"boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/logging"
	"boilerplate-go/internal/observability"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	callSkip = 3
	layer    = "infrastructure"
	pkg      = "driver"

	sqlExec        = "Execute statement"
	sqlQuery       = "Query multiple rows"
	sqlQuerySingle = "Query single row"

	execFunc     = "Exec"
	queryFunc    = "Query"
	queryRowFunc = "QueryRow"
)

// dbWithLogging は DBTX をラップしてログを出してから実処理へ委譲する。
type dbWithLogging struct {
	db       driver.DBTX
	ctx      context.Context
	provider DBProvider
}

func (dwl *dbWithLogging) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	start := time.Now()

	tc, _, end := observability.StartSpanWithParent(ctx, dwl.provider.LayerTracer(), observability.BuildSpanName(layer, pkg, execFunc))
	defer end()

	fields := dwl.buildSQLStartLogFields(tc, execFunc)
	dwl.provider.Logger().Named(layer).CallerSkip(callSkip).Info(sqlExec, fields...)

	res, err := dwl.db.Exec(ctx, sql, args...)
	duration := time.Since(start)

	fields = dwl.buildSQLEndLogFields(tc, execFunc, sql, duration, args, err)
	dwl.logQueryResult(sqlExec, duration, fields, err)
	return res, err
}

func (dwl *dbWithLogging) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	start := time.Now()

	tc, _, end := observability.StartSpanWithParent(ctx, dwl.provider.LayerTracer(), observability.BuildSpanName(layer, pkg, queryFunc))
	defer end()

	fields := dwl.buildSQLStartLogFields(tc, queryFunc)
	dwl.provider.Logger().Named(layer).CallerSkip(callSkip).Info(sqlQuery, fields...)

	rows, err := dwl.db.Query(ctx, sql, args...) //nolint:sqlclosecheck // ownership transferred to caller; closed in sqlc layer
	if err != nil {
		return nil, err
	}
	duration := time.Since(start)

	fields = dwl.buildSQLEndLogFields(tc, queryFunc, sql, duration, args, err)
	dwl.logQueryResult(sqlQuery, duration, fields, err)
	return rows, err
}

func (dwl *dbWithLogging) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	start := time.Now()

	tc, _, end := observability.StartSpanWithParent(ctx, dwl.provider.LayerTracer(), observability.BuildSpanName(layer, pkg, queryRowFunc))
	defer end()

	fields := dwl.buildSQLStartLogFields(tc, queryRowFunc)
	dwl.provider.Logger().Named(layer).CallerSkip(callSkip).Info(sqlQuerySingle, fields...)

	row := dwl.db.QueryRow(ctx, query, args...)
	duration := time.Since(start)

	fields = dwl.buildSQLEndLogFields(tc, queryRowFunc, query, duration, args, nil)
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
	logArgs := args
	if dwl.provider.ObservabilityConfig().MaskedDBQueryArgs() {
		logArgs = nil
	}

	sqlIn := logging.SQLFieldsEndInput{
		Layer:    layer,
		PkgName:  pkg,
		FuncName: funcName,
		SpanName: observability.BuildSpanName(layer, pkg, funcName),

		EventAt: time.Now(),

		Query:        query,
		Latency:      duration,
		Args:         logArgs,
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
