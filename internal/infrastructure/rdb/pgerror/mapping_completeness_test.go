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

func Test_NormalizeError(t *testing.T) {
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

		t.Run("pgx.ErrNoRowsはErrNotFoundへ写像する", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, NormalizeError(pgx.ErrNoRows), apperror.ErrNotFound)
		})

		t.Run("SQLSTATEマップに一致するPgErrorは対応するsentinelへ写像する", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, NormalizeError(&pgconn.PgError{Code: "23505"}), apperror.ErrConflict)
		})

		t.Run("context.CanceledはErrCanceledへ写像する", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, NormalizeError(context.Canceled), apperror.ErrCanceled)
		})

		t.Run("接続不可エラーはErrUnavailableへ写像する", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, NormalizeError(context.DeadlineExceeded), apperror.ErrUnavailable)
		})

		t.Run("分類不能なエラーはErrInternalへ写像する", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, NormalizeError(xerrors.New("boom")), apperror.ErrInternal)
		})
	})
}
