package driver

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/pkg/xerrors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

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

func Test_classifyOperation(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("select", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, operationSelect, classifyOperation("SELECT * FROM users")) //nolint:unqueryvet // 分類器のテスト入力であり実行クエリではない
		})

		t.Run("小文字 select", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, operationSelect, classifyOperation("select 1"))
		})

		t.Run("WITH句はselectに丸める", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, operationSelect, classifyOperation("WITH t AS (SELECT 1) SELECT * FROM t")) //nolint:unqueryvet // 分類器のテスト入力であり実行クエリではない
		})

		t.Run("insert", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, operationInsert, classifyOperation("INSERT INTO users (id) VALUES ($1)"))
		})

		t.Run("update", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, operationUpdate, classifyOperation("UPDATE users SET name = $1"))
		})

		t.Run("delete", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, operationDelete, classifyOperation("DELETE FROM users WHERE id = $1"))
		})

		t.Run("begin", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, operationBegin, classifyOperation("BEGIN"))
		})

		t.Run("start transaction", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, operationBegin, classifyOperation("START TRANSACTION"))
		})

		t.Run("commit", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, operationCommit, classifyOperation("COMMIT"))
		})

		t.Run("rollback", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, operationRollback, classifyOperation("ROLLBACK"))
		})

		t.Run("copy", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, operationCopy, classifyOperation("COPY users FROM STDIN"))
		})

		t.Run("先頭行コメントを無視する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, operationSelect, classifyOperation("-- name: FindUser\nSELECT 1"))
		})

		t.Run("先頭ブロックコメントを無視する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, operationUpdate, classifyOperation("/* hint */ UPDATE users SET x = 1"))
		})

		t.Run("前後空白と括弧を無視する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, operationSelect, classifyOperation("  ( SELECT 1 )"))
		})

		t.Run("不明な先頭トークンはotherに丸める", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, operationOther, classifyOperation("EXPLAIN ANALYZE SELECT 1"))
		})

		t.Run("空文字はotherに丸める", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, operationOther, classifyOperation(""))
		})

		t.Run("閉じないブロックコメントはotherに丸める", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, operationOther, classifyOperation("/* unterminated"))
		})

		t.Run("改行なし行コメントはotherに丸める", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, operationOther, classifyOperation("-- only a comment"))
		})
	})
}

func Test_classifyErrorClass(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilは空文字", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, classifyErrorClass(nil))
		})

		t.Run("SQLSTATE23xxxはconstraint", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, errorClassConstraint, classifyErrorClass(&pgconn.PgError{Code: "23505"}))
		})

		t.Run("lock_timeoutはtimeout", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, errorClassTimeout, classifyErrorClass(&pgconn.PgError{Code: "55P03"}))
		})

		t.Run("statement_timeoutはtimeout", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, errorClassTimeout, classifyErrorClass(&pgconn.PgError{Code: "57014"}))
		})

		t.Run("DeadlineExceededはtimeout", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, errorClassTimeout, classifyErrorClass(context.DeadlineExceeded))
		})

		t.Run("接続例外08xxxはconnection", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, errorClassConnection, classifyErrorClass(&pgconn.PgError{Code: "08006"}))
		})

		t.Run("serialization_failure(40001)はretryable", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, errorClassRetryable, classifyErrorClass(&pgconn.PgError{Code: "40001"}))
		})

		t.Run("deadlock_detected(40P01)はretryable", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, errorClassRetryable, classifyErrorClass(&pgconn.PgError{Code: "40P01"}))
		})

		t.Run("分類不能はunknown", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, errorClassUnknown, classifyErrorClass(xerrors.New("boom")))
		})
	})
}

func Test_buildQueryAttrs(t *testing.T) {
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

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("失敗時はstatus=errorでerror_classが入る", func(t *testing.T) {
			t.Parallel()

			got := buildQueryAttrs(context.Background(), "INSERT INTO users", time.Millisecond, &pgconn.PgError{Code: "23505"})

			assert.Equal(t, operationInsert, got.Operation)
			assert.Equal(t, statusError, got.Status)
			assert.Equal(t, errorClassConstraint, got.ErrorClass)
		})
	})
}
