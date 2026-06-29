//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE
package driver

import (
	"context"
	"errors"
	"strings"
	"time"

	"go-boilerplate/internal/infrastructure/rdb/pgerror"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// query metrics ラベルを低カーディナリティに保つための固定値です。
const (
	queryNameUnknown = "unknown"

	operationSelect   = "select"
	operationInsert   = "insert"
	operationUpdate   = "update"
	operationDelete   = "delete"
	operationBegin    = "begin"
	operationCommit   = "commit"
	operationRollback = "rollback"
	operationCopy     = "copy"
	operationOther    = "other"

	statusSuccess = "success"
	statusError   = "error"

	errorClassConstraint = "constraint"
	errorClassConnection = "connection"
	errorClassTimeout    = "timeout"
	errorClassRetryable  = "retryable"
	errorClassUnknown    = "unknown"
)

// QueryAttrs は、1 クエリの観測属性です。
// SQL 本文・bind 値・テーブル名・制約名・PII などの高カーディナリティ／秘匿情報は持ちません。
type QueryAttrs struct {
	QueryName  string
	Operation  string
	Status     string
	ErrorClass string
	Duration   time.Duration
}

// QueryRecorder は、1 クエリの実行結果をメトリクスとして記録します。
// （interface を消費側の driver に置くことで metrics → driver の循環 import を避けます）。
type QueryRecorder interface {
	Observe(ctx context.Context, attrs QueryAttrs)
}

// queryNameKey は、query_name を context に伝搬するためのキーです。
type queryNameKey struct{}

// WithQueryName は、メトリクスの query_name ラベルに使う安定名を context に付与します。
// Repository / QueryService 側で操作名（例: "user.find_by_id"）を明示します。
func WithQueryName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, queryNameKey{}, name)
}

// queryNameFromContext は、context から query_name を取り出します。
// 未設定または空文字の場合は "unknown" に丸めます。
func queryNameFromContext(ctx context.Context) string {
	name, ok := ctx.Value(queryNameKey{}).(string)
	if !ok || name == "" {
		return queryNameUnknown
	}
	return name
}

// buildQueryAttrs は、クエリ終了時の情報から QueryAttrs を組み立てます。
// pgx.ErrNoRows は QueryRow の通常系（not_found）であり、status=success として扱い error には数えません。
func buildQueryAttrs(ctx context.Context, sql string, duration time.Duration, err error) QueryAttrs {
	status := statusSuccess
	errorClass := ""
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		status = statusError
		errorClass = classifyErrorClass(err)
	}

	return QueryAttrs{
		QueryName:  queryNameFromContext(ctx),
		Operation:  classifyOperation(sql),
		Status:     status,
		ErrorClass: errorClass,
		Duration:   duration,
	}
}

// classifyOperation は、SQL の先頭トークンのみを見て低カーディナリティな operation へ分類します。
// SQL 本文・テーブル名は一切ラベルに含めません。先頭コメントや WITH 句は select / other に丸めます。
func classifyOperation(sql string) string {
	switch firstSQLToken(sql) {
	case "select", "with":
		return operationSelect
	case "insert":
		return operationInsert
	case "update":
		return operationUpdate
	case "delete":
		return operationDelete
	case "begin", "start":
		return operationBegin
	case "commit":
		return operationCommit
	case "rollback":
		return operationRollback
	case "copy":
		return operationCopy
	default:
		return operationOther
	}
}

// firstSQLToken は、先頭コメントと記号を取り除いた最初のトークンを小文字で返します。
func firstSQLToken(sql string) string {
	sql = stripLeadingSQLComments(sql)
	sql = strings.TrimLeft(sql, " \t\r\n(")

	end := strings.IndexFunc(sql, func(r rune) bool {
		switch r {
		case ' ', '\t', '\r', '\n', '(', ';':
			return true
		default:
			return false
		}
	})
	if end >= 0 {
		sql = sql[:end]
	}

	return strings.ToLower(sql)
}

// stripLeadingSQLComments は、SQL 先頭の行コメント(--)・ブロックコメント(/* */)を取り除きます。
func stripLeadingSQLComments(sql string) string {
	for {
		sql = strings.TrimLeft(sql, " \t\r\n")
		switch {
		case strings.HasPrefix(sql, "--"):
			i := strings.IndexByte(sql, '\n')
			if i < 0 {
				return ""
			}
			sql = sql[i+1:]
		case strings.HasPrefix(sql, "/*"):
			i := strings.Index(sql, "*/")
			if i < 0 {
				return ""
			}
			sql = sql[i+2:]
		default:
			return sql
		}
	}
}

// classifyErrorClass は、pgerror の判定を用いてエラーを固定 enum へ丸めます。
// raw error message / SQLSTATE 詳細 / 制約名などはラベルに含めません。
// pgx.ErrNoRows は呼び出し前に除外して渡すこと（success として扱うため）。
func classifyErrorClass(err error) string {
	switch {
	case err == nil:
		return ""
	case isConstraintViolation(err):
		return errorClassConstraint
	case isTimeout(err):
		return errorClassTimeout
	case pgerror.IsRetryableTxError(err):
		return errorClassRetryable
	case pgerror.IsUnavailable(err):
		return errorClassConnection
	default:
		return errorClassUnknown
	}
}

// isConstraintViolation は、整合性制約違反(SQLSTATE 23xxx)であるかを判定します。
func isConstraintViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && strings.HasPrefix(pgErr.Code, "23")
}

// isTimeout は、タイムアウト系のエラー（context 期限切れ / lock_timeout 失効 / statement_timeout）であるかを判定します。
// SQLSTATE 57014(query_canceled) は statement_timeout 失効でも発生するため timeout に分類します。
func isTimeout(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) ||
		pgerror.IsLockNotAvailable(err) ||
		isStatementTimeout(err)
}

// isStatementTimeout は、statement_timeout 失効(SQLSTATE 57014 query_canceled)であるかを判定します。
func isStatementTimeout(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "57014"
}
