package loggingdb

import (
	"context"
	"time"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	// callSkip は、開始ログ(logQueryStart 経由)・終了ログ(logQueryResult 経由)のどちらからでも
	// 呼び出し元 repository 層を caller として記録するためのスキップ段数。
	// 内訳: ヘルパ(1) → Exec/Query/QueryRow(2) → sqlc gen(3) → repository(4)。
	callSkip = 4
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
	provider DBProvider
}

// logger は、SQL ログ用に layer 名と callSkip を設定したロガーを返します。
func (dwl *dbWithLogging) logger() logging.Logger {
	return dwl.provider.Logger().Named(layer).CallerSkip(callSkip)
}

func (dwl *dbWithLogging) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	start := time.Now()

	tc, _, end := observability.StartSpanWithParent(ctx, dwl.provider.LayerTracer(), observability.BuildSpanName(layer, pkg, execFunc))
	defer end()

	dwl.logQueryStart(tc, execFunc, sqlExec)

	res, err := dwl.db.Exec(ctx, sql, args...)
	duration := time.Since(start)

	fields := dwl.buildSQLEndLogFields(tc, execFunc, sql, duration, args, err)
	dwl.logQueryResult(sqlExec, duration, fields, err)
	return res, err
}

func (dwl *dbWithLogging) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	start := time.Now()

	tc, _, end := observability.StartSpanWithParent(ctx, dwl.provider.LayerTracer(), observability.BuildSpanName(layer, pkg, queryFunc))
	defer end()

	dwl.logQueryStart(tc, queryFunc, sqlQuery)

	rows, err := dwl.db.Query(ctx, sql, args...) //nolint:sqlclosecheck // ownership transferred to caller; closed in sqlc layer
	duration := time.Since(start)

	fields := dwl.buildSQLEndLogFields(tc, queryFunc, sql, duration, args, err)
	dwl.logQueryResult(sqlQuery, duration, fields, err)

	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (dwl *dbWithLogging) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	start := time.Now()

	tc, _, end := observability.StartSpanWithParent(ctx, dwl.provider.LayerTracer(), observability.BuildSpanName(layer, pkg, queryRowFunc))
	defer end()

	dwl.logQueryStart(tc, queryRowFunc, sqlQuerySingle)

	row := dwl.db.QueryRow(ctx, query, args...)
	duration := time.Since(start)

	// pgx の QueryRow はエラーを Scan 時まで遅延するため、本層では err を捕捉できず常に nil を渡す。
	// QueryRow 経由の DB エラーは Error レベルでは記録されない（slow-query の Warn のみ機能する）。
	fields := dwl.buildSQLEndLogFields(tc, queryRowFunc, query, duration, args, nil)
	dwl.logQueryResult(sqlQuerySingle, duration, fields, nil)
	return row
}

// logQueryStart は、SQLクエリの開始ログを出力します。
// 終了側 logQueryResult と同じフレーム段数に揃えるため、開始ログも本ヘルパ経由で出力する（callSkip の対称化）。
func (dwl *dbWithLogging) logQueryStart(tc *observability.TraceContext, funcName, msg string) {
	fields := dwl.buildSQLStartLogFields(tc, funcName)
	dwl.logger().Info(msg, fields...)
}

// buildSQLStartLogFields は、SQLクエリの開始ログ出力用フィールドを構築します。
func (dwl *dbWithLogging) buildSQLStartLogFields(tc *observability.TraceContext, funcName string) []*logging.Field {
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
	tc *observability.TraceContext, funcName, query string, duration time.Duration, args []any, err error,
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
	logger := dwl.logger()
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
