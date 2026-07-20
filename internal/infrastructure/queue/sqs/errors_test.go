package sqs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

func Test_normalizeError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilはnilを返す", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, normalizeError(nil))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("context_CanceledはErrCanceledへ正規化する", func(t *testing.T) {
			t.Parallel()

			err := normalizeError(context.Canceled)

			require.ErrorIs(t, err, apperror.ErrCanceled)
			require.ErrorIs(t, err, context.Canceled) // 原因も保持する
		})

		t.Run("その他のbroker由来エラーはErrUnavailableへ正規化する", func(t *testing.T) {
			t.Parallel()

			cause := xerrors.New("throttled")
			err := normalizeError(cause)

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			require.ErrorIs(t, err, cause)
		})
	})
}
