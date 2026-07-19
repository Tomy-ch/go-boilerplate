package driver

import (
	"context"
	"net/url"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/config"
	"go-boilerplate/pkg/xerrors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeQueryRecorder は、recordQueryMetric の呼び出し回数を記録するテスト用 QueryRecorder です。
type fakeQueryRecorder struct {
	observed int
}

func (f *fakeQueryRecorder) Observe(_ context.Context, _ QueryAttrs) {
	f.observed++
}

func Test_buildDSN(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("extraがnilの場合はsslmodeのみのクエリになる", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			dbCfg := config.NewDatabaseConfig(cfg)

			actual := buildDSN(dbCfg, nil)
			require.NotNil(t, actual)

			q := actual.Query()
			assert.Equal(t, dbCfg.SSLMode(), q.Get("sslmode"))
			assert.NotContains(t, actual.RawQuery, "timezone")
		})

		t.Run("extraのクエリパラメータがsslmodeに加えて付与される", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			dbCfg := config.NewDatabaseConfig(cfg)

			actual := buildDSN(dbCfg, url.Values{"timezone": {"Asia/Tokyo"}})
			require.NotNil(t, actual)

			assert.Equal(t, "postgres", actual.Scheme)
			assert.Equal(t, dbCfg.DBName(), actual.Path)
			q := actual.Query()
			assert.Equal(t, dbCfg.SSLMode(), q.Get("sslmode"))
			assert.Equal(t, "Asia/Tokyo", q.Get("timezone"))
		})
	})
}

func Test_newDB(t *testing.T) {
	t.Parallel()
	t.Skip("TestNewDB / TestNewTracedDB（driver パッケージ）の実 DB テストでカバー")
}

func Test_queryNameFromContext(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定済みの query_name をそのまま返す", func(t *testing.T) {
			t.Parallel()

			ctx := WithQueryName(context.Background(), "user.find_by_id")
			assert.Equal(t, "user.find_by_id", queryNameFromContext(ctx))
		})

		t.Run("未設定の場合はunknownに丸める", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, queryNameUnknown, queryNameFromContext(context.Background()))
		})

		t.Run("空文字が設定されている場合はunknownに丸める", func(t *testing.T) {
			t.Parallel()

			ctx := WithQueryName(context.Background(), "")
			assert.Equal(t, queryNameUnknown, queryNameFromContext(ctx))
		})
	})
}

func Test_firstSQLToken(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("先頭トークンを小文字で返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "select", firstSQLToken("SELECT 1"))
		})

		t.Run("先頭コメントと空白と括弧を取り除いた最初のトークンを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "update", firstSQLToken("-- name: X\n  ( UPDATE users SET x = 1"))
		})

		t.Run("セミコロンを区切りとみなす", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "begin", firstSQLToken("BEGIN;"))
		})

		t.Run("空文字は空文字を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, firstSQLToken(""))
		})
	})
}

func Test_stripLeadingSQLComments(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("先頭の行コメントを取り除く", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "SELECT 1", stripLeadingSQLComments("-- name: X\nSELECT 1"))
		})

		t.Run("先頭のブロックコメントを取り除く", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "UPDATE users", stripLeadingSQLComments("/* hint */ UPDATE users"))
		})

		t.Run("複数の先頭コメントを連続して取り除く", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "SELECT 1", stripLeadingSQLComments("-- a\n/* b */ SELECT 1"))
		})

		t.Run("コメントが無い場合はそのまま返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "SELECT 1", stripLeadingSQLComments("SELECT 1"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("閉じない行コメント(改行なし)は空文字を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, stripLeadingSQLComments("-- only a comment"))
		})

		t.Run("閉じないブロックコメントは空文字を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, stripLeadingSQLComments("/* unterminated"))
		})
	})
}

func Test_normalizeTxResult(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilはnilを返す", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, normalizeTxResult(nil))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("PgErrorはapperrorへ正規化される", func(t *testing.T) {
			t.Parallel()
			got := normalizeTxResult(&pgconn.PgError{Code: "23505"})
			require.ErrorIs(t, got, apperror.ErrConflict)
		})

		t.Run("接続不可エラー(08xxx)はapperrorへ正規化される", func(t *testing.T) {
			t.Parallel()
			got := normalizeTxResult(&pgconn.PgError{Code: "08006"})
			require.ErrorIs(t, got, apperror.ErrUnavailable)
		})

		t.Run("context.Canceledはapperrorへ正規化される", func(t *testing.T) {
			t.Parallel()
			got := normalizeTxResult(context.Canceled)
			require.ErrorIs(t, got, apperror.ErrCanceled)
		})

		t.Run("context.DeadlineExceededはapperrorへ正規化される", func(t *testing.T) {
			t.Parallel()
			got := normalizeTxResult(context.DeadlineExceeded)
			require.ErrorIs(t, got, apperror.ErrUnavailable)
		})

		t.Run("fnが返した非DBエラーは正規化せず素通しする", func(t *testing.T) {
			t.Parallel()
			appErr := xerrors.Wrap(apperror.ErrValidation, "boom")
			got := normalizeTxResult(appErr)
			require.ErrorIs(t, got, apperror.ErrValidation)
		})
	})
}

func Test_txManager_doOnce(t *testing.T) {
	t.Parallel()
	t.Skip("Test_txManager_Do（driver_test パッケージ）の実 DB / mock テストでカバー")
}

func Test_txManager_rollback(t *testing.T) {
	t.Parallel()
	t.Skip("Test_txManager_Do の「rollback失敗時はエラーログを出力し元のエラーを返す」ケースでカバー")
}

func Test_queryTracer_recordQueryMetric(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("recorderが設定されている場合はObserveが呼ばれる", func(t *testing.T) {
			t.Parallel()

			rec := &fakeQueryRecorder{}
			qt := &queryTracer{recorder: rec}
			qt.recordQueryMetric(context.Background(), "SELECT 1", time.Millisecond, nil)

			assert.Equal(t, 1, rec.observed)
		})

		t.Run("recorderがnilの場合は何もしない", func(t *testing.T) {
			t.Parallel()

			qt := &queryTracer{recorder: nil}
			// nil recorder でも panic せず no-op で終了する。
			qt.recordQueryMetric(context.Background(), "SELECT 1", time.Millisecond, nil)
		})
	})
}

func Test_queryTracer_endFields(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		ld := queryLogData{sql: "SELECT $1", args: []any{"secret"}, start: time.Now()}

		t.Run("maskArgsがfalseの場合はargs件数フィールドを含むフィールド列を返す", func(t *testing.T) {
			t.Parallel()

			qt, _ := newTestQueryTracer(t)
			qt.maskArgs = false
			plain := qt.endFields(ld, time.Millisecond, nil)

			qt.maskArgs = true
			masked := qt.endFields(ld, time.Millisecond, nil)

			// マスク時は args を nil にするため args_count フィールドが 1 つ減る。
			assert.Len(t, masked, len(plain)-1)
		})
	})
}
