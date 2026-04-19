package pgerror

import (
	"context"
	"errors"
	"testing"

	"go-boilerplate/internal/apperror"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type mockNetError struct{}

func (m mockNetError) Error() string   { return "mock net error" }
func (m mockNetError) Timeout() bool   { return true }
func (m mockNetError) Temporary() bool { return false }

func TestNormalizePgError(t *testing.T) {
	t.Parallel()

	t.Run("errorがnilの場合", func(t *testing.T) {
		t.Parallel()
		got := NormalizeError(nil)
		require.NoError(t, got)
	})

	t.Run("contextが期限切れの場合", func(t *testing.T) {
		t.Parallel()
		got := NormalizeError(context.DeadlineExceeded)
		require.Error(t, got)
		require.ErrorIs(t, got, apperror.ErrUnavailable)
	})

	t.Run("ユニーク制約違反", func(t *testing.T) {
		t.Parallel()
		got := NormalizeError(&pgconn.PgError{Code: "23505", Message: "dup"})
		require.Error(t, got)
		require.ErrorIs(t, got, apperror.ErrConflict)
	})

	t.Run("外部キー制約違反", func(t *testing.T) {
		t.Parallel()
		got := NormalizeError(&pgconn.PgError{Code: "23503", Message: "fk"})
		require.Error(t, got)
		require.ErrorIs(t, got, apperror.ErrInvalidArgument)
	})

	t.Run("NOT NULL制約違反", func(t *testing.T) {
		t.Parallel()
		got := NormalizeError(&pgconn.PgError{Code: "23502", Message: "not null"})
		require.Error(t, got)
		require.ErrorIs(t, got, apperror.ErrInvalidArgument)
	})

	t.Run("チェック制約違反", func(t *testing.T) {
		t.Parallel()
		got := NormalizeError(&pgconn.PgError{Code: "23514", Message: "check constraint"})
		require.Error(t, got)
		require.ErrorIs(t, got, apperror.ErrInvalidArgument)
	})

	t.Run("文字数超過", func(t *testing.T) {
		t.Parallel()
		got := NormalizeError(&pgconn.PgError{Code: "22001", Message: "too long"})
		require.Error(t, got)
		require.ErrorIs(t, got, apperror.ErrInvalidArgument)
	})

	t.Run("型変換エラー", func(t *testing.T) {
		t.Parallel()
		got := NormalizeError(&pgconn.PgError{Code: "22P02", Message: "bad input"})
		require.Error(t, got)
		require.ErrorIs(t, got, apperror.ErrInvalidArgument)
	})

	t.Run("権限不足", func(t *testing.T) {
		t.Parallel()
		got := NormalizeError(&pgconn.PgError{Code: "42501", Message: "no permission"})
		require.Error(t, got)
		require.ErrorIs(t, got, apperror.ErrPermissionDenied)
	})

	t.Run("直列化失敗", func(t *testing.T) {
		t.Parallel()
		got := NormalizeError(&pgconn.PgError{Code: "40001", Message: "serialization failure"})
		require.Error(t, got)
		require.ErrorIs(t, got, apperror.ErrUnavailable)
	})

	t.Run("トランザクションのデッドロック", func(t *testing.T) {
		t.Parallel()
		got := NormalizeError(&pgconn.PgError{Code: "40P01", Message: "transaction failure"})
		require.Error(t, got)
		require.ErrorIs(t, got, apperror.ErrUnavailable)
	})

	t.Run("クエリのキャンセル", func(t *testing.T) {
		t.Parallel()
		got := NormalizeError(&pgconn.PgError{Code: "57014", Message: "query canceled"})
		require.Error(t, got)
		require.ErrorIs(t, got, apperror.ErrUnavailable)
	})

	t.Run("該当なし（NoRows）", func(t *testing.T) {
		t.Parallel()
		got := NormalizeError(pgx.ErrNoRows)
		require.Error(t, got)
		require.ErrorIs(t, got, apperror.ErrNotFound)
	})

	t.Run("Postgres接続エラー(08xxx)", func(t *testing.T) {
		t.Parallel()
		got := NormalizeError(&pgconn.PgError{Code: "08006", Message: "conn"})
		require.Error(t, got)
		require.ErrorIs(t, got, apperror.ErrUnavailable)
	})

	t.Run("不明なPostgresエラー", func(t *testing.T) {
		t.Parallel()
		got := NormalizeError(&pgconn.PgError{Code: "99999", Message: "unknown"})
		require.Error(t, got)
		require.ErrorIs(t, got, apperror.ErrInternal)
	})

	t.Run("その他のエラー", func(t *testing.T) {
		t.Parallel()
		got := NormalizeError(errors.New("generic"))
		require.Error(t, got)
		require.ErrorIs(t, got, apperror.ErrInternal)
	})
}

func TestIsUnavailable(t *testing.T) {
	t.Parallel()
	t.Run("nil", func(t *testing.T) {
		t.Parallel()
		got := IsUnavailable(nil)
		require.False(t, got)
	})

	t.Run("コンテキスト期限切れ", func(t *testing.T) {
		t.Parallel()
		got := IsUnavailable(context.DeadlineExceeded)
		require.True(t, got)
	})

	t.Run("ネットワークエラー", func(t *testing.T) {
		t.Parallel()
		got := IsUnavailable(mockNetError{})
		require.True(t, got)
	})

	t.Run("Postgres接続エラー", func(t *testing.T) {
		t.Parallel()
		got := IsUnavailable(&pgconn.PgError{Code: "08003"})
		require.True(t, got)
	})

	t.Run("Postgres非接続エラー", func(t *testing.T) {
		t.Parallel()
		got := IsUnavailable(&pgconn.PgError{Code: "23505"})
		require.False(t, got)
	})

	t.Run("その他のエラー", func(t *testing.T) {
		t.Parallel()
		got := IsUnavailable(errors.New("other"))
		require.False(t, got)
	})
}

func Test_isPgConnectionError(t *testing.T) {
	t.Parallel()
	t.Run("Postgres接続エラー", func(t *testing.T) {
		t.Parallel()
		got := isPgConnectionError(&pgconn.PgError{Code: "08006"})
		require.True(t, got)
	})

	t.Run("Postgres非接続エラー", func(t *testing.T) {
		t.Parallel()
		got := isPgConnectionError(&pgconn.PgError{Code: "23505"})
		require.False(t, got)
	})

	t.Run("その他のエラー", func(t *testing.T) {
		t.Parallel()
		got := isPgConnectionError(errors.New("other"))
		require.False(t, got)
	})
}
