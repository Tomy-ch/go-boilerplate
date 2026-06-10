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
	// callSkip は呼び出し元 repository 層を caller に記録するための段数（ヘルパ→メソッド→sqlc gen→repository）。
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
	provider *provider
}

func (dwl *dbWithLogging) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	start := time.Now()

	spanName := observability.BuildSpanName(layer, pkg, execFunc)
	tc, _, end := observability.StartSpanWithParent(ctx, dwl.provider.tracer, spanName)
	defer end()

	dwl.logQueryStart(tc, execFunc, spanName, sqlExec)

	res, err := dwl.db.Exec(ctx, sql, args...)
	duration := time.Since(start)

	fields := dwl.buildSQLEndLogFields(tc, execFunc, spanName, sql, duration, args, err)
	dwl.logQueryResult(sqlExec, duration, fields, err)
	return res, err
}

func (dwl *dbWithLogging) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	start := time.Now()

	spanName := observability.BuildSpanName(layer, pkg, queryFunc)
	tc, _, end := observability.StartSpanWithParent(ctx, dwl.provider.tracer, spanName)
	defer end()

	dwl.logQueryStart(tc, queryFunc, spanName, sqlQuery)

	rows, err := dwl.db.Query(ctx, sql, args...) //nolint:sqlclosecheck // ownership transferred to caller; closed in sqlc layer
	duration := time.Since(start)

	fields := dwl.buildSQLEndLogFields(tc, queryFunc, spanName, sql, duration, args, err)
	dwl.logQueryResult(sqlQuery, duration, fields, err)

	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (dwl *dbWithLogging) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	start := time.Now()

	spanName := observability.BuildSpanName(layer, pkg, queryRowFunc)
	tc, _, end := observability.StartSpanWithParent(ctx, dwl.provider.tracer, spanName)
	defer end()

	dwl.logQueryStart(tc, queryRowFunc, spanName, sqlQuerySingle)

	row := dwl.db.QueryRow(ctx, query, args...)
	duration := time.Since(start)

	// pgx の QueryRow はエラーを Scan 時まで遅延するため本層では捕捉できず、Error ログは出ない（slow-query の Warn のみ）。
	fields := dwl.buildSQLEndLogFields(tc, queryRowFunc, spanName, query, duration, args, nil)
	dwl.logQueryResult(sqlQuerySingle, duration, fields, nil)
	return row
}

// logger は layer 名と callSkip を設定したロガーを返す。
func (dwl *dbWithLogging) logger() logging.Logger {
	return dwl.provider.l.Named(layer).CallerSkip(callSkip)
}

// logQueryStart は終了側 logQueryResult とフレーム段数を揃えるためのヘルパ（callSkip の対称化）。
func (dwl *dbWithLogging) logQueryStart(tc *observability.TraceContext, funcName, spanName, msg string) {
	fields := dwl.buildSQLStartLogFields(tc, funcName, spanName)
	dwl.logger().Info(msg, fields...)
}

// buildSQLStartLogFields は、SQLクエリの開始ログ出力用フィールドを構築します。
func (dwl *dbWithLogging) buildSQLStartLogFields(tc *observability.TraceContext, funcName, spanName string) []*logging.Field {
	sqlIn := logging.SQLFieldsStartInput{
		Layer:    layer,
		PkgName:  pkg,
		FuncName: funcName,
		SpanName: spanName,

		EventAt: time.Now(),

		TraceID:      tc.TraceID(),
		SpanID:       tc.SpanID(),
		ParentSpanID: tc.ParentSpanID(),
	}
	return dwl.provider.lf.BuildSQLStartFields(sqlIn)
}

// buildSQLEndLogFields は、SQLクエリの終了ログ出力用フィールドを構築します。
func (dwl *dbWithLogging) buildSQLEndLogFields(
	tc *observability.TraceContext, funcName, spanName, query string, duration time.Duration, args []any, err error,
) []*logging.Field {
	logArgs := args
	if dwl.provider.obsCfg.MaskedDBQueryArgs() {
		logArgs = nil
	}

	sqlIn := logging.SQLFieldsEndInput{
		Layer:    layer,
		PkgName:  pkg,
		FuncName: funcName,
		SpanName: spanName,

		EventAt: time.Now(),

		Query:        query,
		Latency:      duration,
		Args:         logArgs,
		Err:          err,
		TraceID:      tc.TraceID(),
		SpanID:       tc.SpanID(),
		ParentSpanID: tc.ParentSpanID(),
	}
	return dwl.provider.lf.BuildSQLEndFields(sqlIn)
}

// logQueryResult は、SQLクエリの実行結果をログ出力します。
func (dwl *dbWithLogging) logQueryResult(
	msg string, duration time.Duration, fields []*logging.Field, err error,
) {
	logger := dwl.logger()
	threshold := dwl.provider.dbCfg.SlowQueryWarnThreshold()
	switch {
	case err != nil:
		logger.Error(msg, fields...)
	case threshold > 0 && duration > threshold:
		logger.Warn(msg, fields...)
	default:
		logger.Info(msg, fields...)
	}
}
