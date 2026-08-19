package pgerror

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockNetError struct{}

type mockNonTimeoutNetError struct{}

func (m mockNetError) Error() string   { return "mock net error" }
func (m mockNetError) Timeout() bool   { return true }
func (m mockNetError) Temporary() bool { return false }

func (m mockNonTimeoutNetError) Error() string   { return "connection refused" }
func (m mockNonTimeoutNetError) Timeout() bool   { return false }
func (m mockNonTimeoutNetError) Temporary() bool { return false }

func TestIsUnavailable(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilの場合は接続不可ではない", func(t *testing.T) {
			t.Parallel()
			got := IsUnavailable(nil)
			assert.False(t, got)
		})

		t.Run("コンテキスト期限切れ", func(t *testing.T) {
			t.Parallel()
			got := IsUnavailable(context.DeadlineExceeded)
			assert.True(t, got)
		})

		t.Run("コンテキストキャンセルはクライアント起因なので接続不可ではない", func(t *testing.T) {
			t.Parallel()
			got := IsUnavailable(context.Canceled)
			assert.False(t, got)
		})

		t.Run("ネットワークエラー(タイムアウト)", func(t *testing.T) {
			t.Parallel()
			got := IsUnavailable(mockNetError{})
			assert.True(t, got)
		})

		t.Run("ネットワークエラー(非タイムアウト/接続拒否)", func(t *testing.T) {
			t.Parallel()
			got := IsUnavailable(mockNonTimeoutNetError{})
			assert.True(t, got)
		})

		t.Run("Postgres接続エラー", func(t *testing.T) {
			t.Parallel()
			got := IsUnavailable(&pgconn.PgError{Code: "08003"})
			assert.True(t, got)
		})

		t.Run("Postgres非接続エラー", func(t *testing.T) {
			t.Parallel()
			got := IsUnavailable(&pgconn.PgError{Code: "23505"})
			assert.False(t, got)
		})

		t.Run("その他のエラー", func(t *testing.T) {
			t.Parallel()
			got := IsUnavailable(xerrors.New("other"))
			assert.False(t, got)
		})
	})
}

func Test_isPgConnectionError(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Postgres接続エラー", func(t *testing.T) {
			t.Parallel()
			got := isPgConnectionError(&pgconn.PgError{Code: "08006"})
			assert.True(t, got)
		})

		t.Run("Postgres非接続エラー", func(t *testing.T) {
			t.Parallel()
			got := isPgConnectionError(&pgconn.PgError{Code: "23505"})
			assert.False(t, got)
		})

		t.Run("その他のエラー", func(t *testing.T) {
			t.Parallel()
			got := isPgConnectionError(xerrors.New("other"))
			assert.False(t, got)
		})
	})
}

func TestIsLockNotAvailable(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("55P03(lock_not_available)はtrue", func(t *testing.T) {
			t.Parallel()
			assert.True(t, IsLockNotAvailable(&pgconn.PgError{Code: "55P03"}))
		})

		t.Run("別のSQLSTATEはfalse", func(t *testing.T) {
			t.Parallel()
			assert.False(t, IsLockNotAvailable(&pgconn.PgError{Code: "23505"}))
		})

		t.Run("PgError以外はfalse", func(t *testing.T) {
			t.Parallel()
			assert.False(t, IsLockNotAvailable(xerrors.New("plain error")))
		})
	})
}

func TestIsNoRows(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("pgx.ErrNoRowsはtrue", func(t *testing.T) {
			t.Parallel()
			assert.True(t, IsNoRows(pgx.ErrNoRows))
		})

		t.Run("ラップされたpgx.ErrNoRowsはtrue", func(t *testing.T) {
			t.Parallel()
			assert.True(t, IsNoRows(xerrors.Wrap(pgx.ErrNoRows, "wrapped")))
		})

		t.Run("nilはfalse", func(t *testing.T) {
			t.Parallel()
			assert.False(t, IsNoRows(nil))
		})

		t.Run("行なし以外のエラーはfalse", func(t *testing.T) {
			t.Parallel()
			assert.False(t, IsNoRows(context.Canceled))
		})
	})
}

func TestIsRetryableTxError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("40001(serialization_failure)はtrue", func(t *testing.T) {
			t.Parallel()
			assert.True(t, IsRetryableTxError(&pgconn.PgError{Code: "40001"}))
		})

		t.Run("40P01(deadlock_detected)はtrue", func(t *testing.T) {
			t.Parallel()
			assert.True(t, IsRetryableTxError(&pgconn.PgError{Code: "40P01"}))
		})

		t.Run("別のSQLSTATEはfalse", func(t *testing.T) {
			t.Parallel()
			assert.False(t, IsRetryableTxError(&pgconn.PgError{Code: "23505"}))
		})

		t.Run("PgError以外はfalse", func(t *testing.T) {
			t.Parallel()
			assert.False(t, IsRetryableTxError(xerrors.New("plain error")))
		})
	})
}

func TestNormalizeExecResult(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("影響行数が1以上かつerrorなしの場合、nilを返す", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, NormalizeExecResult(1, nil))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("影響行数が0かつerrorなしの場合、pgx.ErrNoRows由来と区別できるErrNotFoundを返す", func(t *testing.T) {
			t.Parallel()
			got := NormalizeExecResult(0, nil)
			require.ErrorIs(t, got, apperror.ErrNotFound)
			require.NotErrorIs(t, got, pgx.ErrNoRows)
		})

		t.Run("errorがある場合、NormalizeErrorへ委譲し元エラーもチェーンに残す", func(t *testing.T) {
			t.Parallel()
			src := &pgconn.PgError{Code: "23505"}

			got := NormalizeExecResult(0, src)

			require.ErrorIs(t, got, apperror.ErrConflict)
			require.ErrorIs(t, got, src)
		})
	})
}

func TestNormalizeReconstructError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilの場合、nilを返す", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, NormalizeReconstructError(nil))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("検証エラーがErrInternalへ平坦化され元の分類は消える", func(t *testing.T) {
			t.Parallel()
			src := apperror.WithDetails(xerrors.Wrap(apperror.ErrValidation, "first name failed"), "firstName")

			got := NormalizeReconstructError(src)

			require.ErrorIs(t, got, apperror.ErrInternal)
			require.NotErrorIs(t, got, apperror.ErrValidation)
			_, ok := apperror.MetaFrom(got)
			assert.False(t, ok)
		})

		t.Run("理由文はメッセージに保持される", func(t *testing.T) {
			t.Parallel()
			src := xerrors.Wrap(apperror.ErrValidation, "first name failed")

			got := NormalizeReconstructError(src)

			assert.Contains(t, got.Error(), "first name failed")
		})
	})
}

func TestNormalizeError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilはnilを返す", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, NormalizeError(nil))
		})

		t.Run("正規化済みapperrorは分類を保持して素通しする", func(t *testing.T) {
			t.Parallel()
			once := NormalizeError(&pgconn.PgError{Code: "23505"})
			twice := NormalizeError(once)
			require.ErrorIs(t, twice, apperror.ErrConflict)
			assert.Equal(t, once, twice)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("pgx.ErrNoRowsはErrNotFoundへ写像し元エラーもチェーンに残す", func(t *testing.T) {
			t.Parallel()
			got := NormalizeError(pgx.ErrNoRows)
			require.ErrorIs(t, got, apperror.ErrNotFound)
			require.ErrorIs(t, got, pgx.ErrNoRows)
		})

		t.Run("SQLSTATEマップに一致するPgErrorは対応するsentinelへ写像する", func(t *testing.T) {
			t.Parallel()

			t.Run("ユニーク制約違反(23505)はErrConflictへ写像する", func(t *testing.T) {
				t.Parallel()
				require.ErrorIs(t, NormalizeError(&pgconn.PgError{Code: "23505"}), apperror.ErrConflict)
			})

			t.Run("外部キー制約違反(23503)はErrInvalidArgumentへ写像する", func(t *testing.T) {
				t.Parallel()
				require.ErrorIs(t, NormalizeError(&pgconn.PgError{Code: "23503"}), apperror.ErrInvalidArgument)
			})

			t.Run("NOT NULL制約違反(23502)はErrInvalidArgumentへ写像する", func(t *testing.T) {
				t.Parallel()
				require.ErrorIs(t, NormalizeError(&pgconn.PgError{Code: "23502"}), apperror.ErrInvalidArgument)
			})

			t.Run("チェック制約違反(23514)はErrInvalidArgumentへ写像する", func(t *testing.T) {
				t.Parallel()
				require.ErrorIs(t, NormalizeError(&pgconn.PgError{Code: "23514"}), apperror.ErrInvalidArgument)
			})

			t.Run("文字数超過(22001)はErrInvalidArgumentへ写像する", func(t *testing.T) {
				t.Parallel()
				require.ErrorIs(t, NormalizeError(&pgconn.PgError{Code: "22001"}), apperror.ErrInvalidArgument)
			})

			t.Run("符号化できないバイト(22021)はErrInvalidArgumentへ写像する", func(t *testing.T) {
				t.Parallel()
				require.ErrorIs(t, NormalizeError(&pgconn.PgError{Code: "22021"}), apperror.ErrInvalidArgument)
			})

			t.Run("型変換エラー(22P02)はErrInvalidArgumentへ写像する", func(t *testing.T) {
				t.Parallel()
				require.ErrorIs(t, NormalizeError(&pgconn.PgError{Code: "22P02"}), apperror.ErrInvalidArgument)
			})

			t.Run("権限不足(42501)はErrPermissionDeniedへ写像する", func(t *testing.T) {
				t.Parallel()
				require.ErrorIs(t, NormalizeError(&pgconn.PgError{Code: "42501"}), apperror.ErrPermissionDenied)
			})

			t.Run("直列化失敗(40001)はErrUnavailableへ写像する", func(t *testing.T) {
				t.Parallel()
				require.ErrorIs(t, NormalizeError(&pgconn.PgError{Code: "40001"}), apperror.ErrUnavailable)
			})

			t.Run("デッドロック(40P01)はErrUnavailableへ写像する", func(t *testing.T) {
				t.Parallel()
				require.ErrorIs(t, NormalizeError(&pgconn.PgError{Code: "40P01"}), apperror.ErrUnavailable)
			})

			t.Run("ロック待ちのタイムアウト(55P03)はErrUnavailableへ写像する", func(t *testing.T) {
				t.Parallel()
				require.ErrorIs(t, NormalizeError(&pgconn.PgError{Code: "55P03"}), apperror.ErrUnavailable)
			})

			t.Run("クエリのキャンセル(57014)はErrUnavailableへ写像する", func(t *testing.T) {
				t.Parallel()
				require.ErrorIs(t, NormalizeError(&pgconn.PgError{Code: "57014"}), apperror.ErrUnavailable)
			})
		})

		t.Run("SQLSTATEマップに無い接続例外クラス(08006)はErrUnavailableへ写像する", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, NormalizeError(&pgconn.PgError{Code: "08006"}), apperror.ErrUnavailable)
		})

		t.Run("SQLSTATEマップに無く接続例外でもないPgError(99999)はErrInternalへ写像する", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, NormalizeError(&pgconn.PgError{Code: "99999"}), apperror.ErrInternal)
		})

		t.Run("写像後も元のPgErrorをxerrors.Asでチェーンから取り出せる", func(t *testing.T) {
			t.Parallel()
			src := &pgconn.PgError{Code: "23505", Message: "dup"}

			got := NormalizeError(src)

			var pgErr *pgconn.PgError
			require.True(t, xerrors.As(got, &pgErr))
			assert.Same(t, src, pgErr)
		})

		t.Run("context.CanceledはErrCanceledへ写像し元エラーもチェーンに残す", func(t *testing.T) {
			t.Parallel()
			got := NormalizeError(context.Canceled)
			require.ErrorIs(t, got, apperror.ErrCanceled)
			require.ErrorIs(t, got, context.Canceled)
		})

		t.Run("接続不可エラーはErrUnavailableへ写像し元エラーもチェーンに残す", func(t *testing.T) {
			t.Parallel()
			got := NormalizeError(context.DeadlineExceeded)
			require.ErrorIs(t, got, apperror.ErrUnavailable)
			require.ErrorIs(t, got, context.DeadlineExceeded)
		})

		t.Run("分類不能なエラーはErrInternalへ写像し元エラーもチェーンに残す", func(t *testing.T) {
			t.Parallel()
			src := xerrors.New("boom")

			got := NormalizeError(src)

			require.ErrorIs(t, got, apperror.ErrInternal)
			require.ErrorIs(t, got, src)
		})
	})
}
