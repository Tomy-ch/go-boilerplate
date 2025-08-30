package rdbdriver

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"go.uber.org/zap"
)

// dbWithLogging は DBTX をラップしてログを出してから実処理へ委譲する。
type dbWithLogging struct {
	db DBTX
	l  *zap.Logger
}

// NewLoggingDB は、DBTXにログ出力機能を追加します。
func NewLoggingDB(db DBTX, log *zap.Logger) DBTX {
	return &dbWithLogging{db: db, l: log}
}

func (dwl *dbWithLogging) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	start := time.Now()
	res, err := dwl.db.ExecContext(ctx, query, args...)
	cleanQuery := strings.ReplaceAll(query, "\n", " ")

	fields := buildZapWithArgsFields(cleanQuery, time.Since(start), args...)
	if err != nil {
		fields = append(fields, zap.NamedError("error", err))
		dwl.l.Error("SQL Exec", fields...)
	} else {
		dwl.l.Info("SQL Exec", fields...)
	}

	return res, err
}

func (dwl *dbWithLogging) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	start := time.Now()
	stmt, err := dwl.db.PrepareContext(ctx, query)
	cleanQuery := strings.ReplaceAll(query, "\n", " ")

	fields := buildZapFields(cleanQuery, time.Since(start))
	if err != nil {
		fields = append(fields, zap.NamedError("error", err))
		dwl.l.Error("SQL Prepare", fields...)
	} else {
		dwl.l.Info("SQL Prepare", fields...)
	}

	return stmt, err
}

func (dwl *dbWithLogging) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	start := time.Now()
	rows, err := dwl.db.QueryContext(ctx, query, args...)
	cleanQuery := strings.ReplaceAll(query, "\n", " ")

	fields := buildZapWithArgsFields(cleanQuery, time.Since(start), args...)
	if err != nil {
		fields = append(fields, zap.NamedError("error", err))
		dwl.l.Error("SQL Query", fields...)
	} else {
		dwl.l.Info("SQL Query", fields...)
	}

	return rows, err
}

func (dwl *dbWithLogging) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	start := time.Now()
	row := dwl.db.QueryRowContext(ctx, query, args...)
	cleanQuery := strings.ReplaceAll(query, "\n", " ")
	fields := buildZapWithArgsFields(cleanQuery, time.Since(start), args...)
	dwl.l.Debug("SQL QueryRow", fields...)
	return row
}

// buildZapFields は、SQLクエリのログ出力用フィールドを構築します。
func buildZapFields(q string, t time.Duration) []zap.Field {
	sec := float64(t) / float64(time.Second)
	return []zap.Field{
		zap.String("query", q),
		zap.Float64("dur_sec", sec),
	}
}

// buildZapWithArgsFields は、SQLクエリのログ出力用フィールドを構築します。
func buildZapWithArgsFields(q string, t time.Duration, args ...any) []zap.Field {
	fields := buildZapFields(q, t)
	fields = append(fields, zap.Any("args", args))
	return fields
}
