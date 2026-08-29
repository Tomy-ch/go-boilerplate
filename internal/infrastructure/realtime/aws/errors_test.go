package aws

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

func Test_normalize(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("context の取り消しは ErrCanceled", func(t *testing.T) {
			t.Parallel()

			err := normalize(context.Canceled, "op")
			require.ErrorIs(t, err, apperror.ErrCanceled)
			assert.Contains(t, err.Error(), "op: ")
		})

		t.Run("期限切れも ErrCanceled", func(t *testing.T) {
			t.Parallel()

			require.ErrorIs(t, normalize(context.DeadlineExceeded, "op"), apperror.ErrCanceled)
		})

		t.Run("それ以外は ErrUnavailable", func(t *testing.T) {
			t.Parallel()

			err := normalize(xerrors.New("boom"), "op")
			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Contains(t, err.Error(), "boom")
		})

		t.Run("原因は chain に残る", func(t *testing.T) {
			t.Parallel()

			cause := xerrors.New("boom")
			require.ErrorIs(t, normalize(cause, "op"), cause)
		})
	})
}
