package apperror_test

import (
	"errors"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
)

func TestIsAppError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("apperror センチネルそのものの場合 true", func(t *testing.T) {
			t.Parallel()
			assert.True(t, apperror.IsAppError(apperror.ErrNotFound))
		})

		t.Run("apperror をラップしたエラーの場合 true", func(t *testing.T) {
			t.Parallel()
			wrapped := xerrors.Wrap(apperror.ErrConflict, "duplicated")
			assert.True(t, apperror.IsAppError(wrapped))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("apperror に該当しないエラーの場合 false", func(t *testing.T) {
			t.Parallel()
			assert.False(t, apperror.IsAppError(errors.New("generic")))
		})

		t.Run("nil の場合 false", func(t *testing.T) {
			t.Parallel()
			assert.False(t, apperror.IsAppError(nil))
		})
	})
}
