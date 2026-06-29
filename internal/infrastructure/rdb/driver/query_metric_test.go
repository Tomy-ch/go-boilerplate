package driver

import (
	"context"
	"sync"
	"testing"
	"time"

	"go-boilerplate/pkg/xerrors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

// fakeQueryRecorder は、driver パッケージ内テスト用の QueryRecorder 実装です。
// metrics パッケージを import すると循環参照になるため、テスト専用の最小実装を用意します。
type fakeQueryRecorder struct {
	mu       sync.Mutex
	observed []QueryAttrs
}

func (r *fakeQueryRecorder) Observe(_ context.Context, attrs QueryAttrs) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.observed = append(r.observed, attrs)
}

func (r *fakeQueryRecorder) snapshot() []QueryAttrs {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]QueryAttrs(nil), r.observed...)
}

func TestWithQueryName(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定した query_name を取り出せる", func(t *testing.T) {
			t.Parallel()

			ctx := WithQueryName(context.Background(), "user.find_by_id")
			assert.Equal(t, "user.find_by_id", queryNameFromContext(ctx))
		})

		t.Run("未設定の場合は unknown を返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, queryNameUnknown, queryNameFromContext(context.Background()))
		})

		t.Run("空文字を設定した場合は unknown を返す", func(t *testing.T) {
			t.Parallel()

			ctx := WithQueryName(context.Background(), "")
			assert.Equal(t, queryNameUnknown, queryNameFromContext(ctx))
		})
	})
}

func TestClassifyOperation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		sql  string
		want string
	}{
		{name: "select", sql: "SELECT * FROM users", want: operationSelect},
		{name: "小文字 select", sql: "select 1", want: operationSelect},
		{name: "WITH句はselectに丸める", sql: "WITH t AS (SELECT 1) SELECT * FROM t", want: operationSelect},
		{name: "insert", sql: "INSERT INTO users (id) VALUES ($1)", want: operationInsert},
		{name: "update", sql: "UPDATE users SET name = $1", want: operationUpdate},
		{name: "delete", sql: "DELETE FROM users WHERE id = $1", want: operationDelete},
		{name: "begin", sql: "BEGIN", want: operationBegin},
		{name: "start transaction", sql: "START TRANSACTION", want: operationBegin},
		{name: "commit", sql: "COMMIT", want: operationCommit},
		{name: "rollback", sql: "ROLLBACK", want: operationRollback},
		{name: "copy", sql: "COPY users FROM STDIN", want: operationCopy},
		{name: "先頭行コメントを無視する", sql: "-- name: FindUser\nSELECT 1", want: operationSelect},
		{name: "先頭ブロックコメントを無視する", sql: "/* hint */ UPDATE users SET x = 1", want: operationUpdate},
		{name: "前後空白と括弧を無視する", sql: "  ( SELECT 1 )", want: operationSelect},
		{name: "不明な先頭トークンはotherに丸める", sql: "EXPLAIN ANALYZE SELECT 1", want: operationOther},
		{name: "空文字はotherに丸める", sql: "", want: operationOther},
		{name: "閉じないブロックコメントはotherに丸める", sql: "/* unterminated", want: operationOther},
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				assert.Equal(t, tt.want, classifyOperation(tt.sql))
			})
		}
	})
}

func TestClassifyErrorClass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nilは空文字", err: nil, want: ""},
		{name: "ErrNoRowsはnot_found", err: pgx.ErrNoRows, want: errorClassNotFound},
		{name: "SQLSTATE23xxxはconstraint", err: &pgconn.PgError{Code: "23505"}, want: errorClassConstraint},
		{name: "lock_timeoutはtimeout", err: &pgconn.PgError{Code: "55P03"}, want: errorClassTimeout},
		{name: "statement_timeoutはtimeout", err: &pgconn.PgError{Code: "57014"}, want: errorClassTimeout},
		{name: "DeadlineExceededはtimeout", err: context.DeadlineExceeded, want: errorClassTimeout},
		{name: "接続例外08xxxはconnection", err: &pgconn.PgError{Code: "08006"}, want: errorClassConnection},
		{name: "分類不能はunknown", err: xerrors.New("boom"), want: errorClassUnknown},
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				assert.Equal(t, tt.want, classifyErrorClass(tt.err))
			})
		}
	})
}

func TestBuildQueryAttrs(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("成功時はstatus=successでerror_classは空", func(t *testing.T) {
			t.Parallel()

			ctx := WithQueryName(context.Background(), "user.find_by_id")
			got := buildQueryAttrs(ctx, "SELECT 1", 5*time.Millisecond, nil)

			assert.Equal(t, "user.find_by_id", got.QueryName)
			assert.Equal(t, operationSelect, got.Operation)
			assert.Equal(t, statusSuccess, got.Status)
			assert.Empty(t, got.ErrorClass)
			assert.Equal(t, 5*time.Millisecond, got.Duration)
		})

		t.Run("ErrNoRowsはsuccess扱いでerrorに数えない", func(t *testing.T) {
			t.Parallel()

			got := buildQueryAttrs(context.Background(), "SELECT 1", time.Millisecond, pgx.ErrNoRows)

			assert.Equal(t, queryNameUnknown, got.QueryName)
			assert.Equal(t, statusSuccess, got.Status)
			assert.Empty(t, got.ErrorClass)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("失敗時はstatus=errorでerror_classが入る", func(t *testing.T) {
			t.Parallel()

			got := buildQueryAttrs(context.Background(), "INSERT INTO users", time.Millisecond, &pgconn.PgError{Code: "23505"})

			assert.Equal(t, operationInsert, got.Operation)
			assert.Equal(t, statusError, got.Status)
			assert.Equal(t, errorClassConstraint, got.ErrorClass)
		})

		t.Run("ラベルにSQL本文やbind値を含めない", func(t *testing.T) {
			t.Parallel()

			ctx := WithQueryName(context.Background(), "user.find_by_email")
			sql := "SELECT id FROM users WHERE email = $1"
			got := buildQueryAttrs(ctx, sql, time.Millisecond, nil)

			// operation / query_name は固定 enum / 明示名のみで、SQL 本文や bind 値は混入しない。
			assert.NotContains(t, got.Operation, "users")
			assert.NotContains(t, got.Operation, "email")
			assert.NotContains(t, got.QueryName, "SELECT")
			assert.Equal(t, operationSelect, got.Operation)
		})
	})
}
