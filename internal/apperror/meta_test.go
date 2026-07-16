package apperror_test

import (
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithMeta(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("元エラーのセンチネル分類を保持する", func(t *testing.T) {
			t.Parallel()
			err := apperror.WithMeta(xerrors.Wrap(apperror.ErrValidation, "invalid"), apperror.Meta{Code: "CUSTOM"})
			require.ErrorIs(t, err, apperror.ErrValidation)
			assert.True(t, apperror.IsAppError(err))
		})

		t.Run("Join された複数センチネルすべての分類を保持する", func(t *testing.T) {
			t.Parallel()
			joined := xerrors.Join(
				xerrors.Wrap(apperror.ErrValidation, "first name failed"),
				xerrors.Wrap(apperror.ErrValidation, "email failed"),
			)
			err := apperror.WithDetails(joined, "firstName", "email")
			require.ErrorIs(t, err, apperror.ErrValidation)
			assert.True(t, apperror.IsAppError(err))
		})

		t.Run("エラーメッセージは元エラーのまま", func(t *testing.T) {
			t.Parallel()
			err := apperror.WithMeta(xerrors.New("original"), apperror.Meta{Code: "CUSTOM"})
			assert.Equal(t, "original", err.Error())
		})

		t.Run("スタックトレース表現は元エラーへ委譲される", func(t *testing.T) {
			t.Parallel()
			err := apperror.WithMeta(xerrors.Wrap(apperror.ErrValidation, "invalid"), apperror.Meta{})
			assert.Contains(t, xerrors.StackTrace(err), "meta_test.go")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nil の場合 nil を返す", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, apperror.WithMeta(nil, apperror.Meta{Code: "CUSTOM"}))
		})
	})
}

func TestMetaFrom(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("付与した Code / Message / Details を抽出できる", func(t *testing.T) {
			t.Parallel()
			err := apperror.WithMeta(apperror.ErrValidation, apperror.Meta{
				Code:    "CUSTOM_CODE",
				Message: "custom message",
				Details: []string{"firstName", "email"},
			})
			meta, ok := apperror.MetaFrom(err)
			require.True(t, ok)
			assert.Equal(t, "CUSTOM_CODE", meta.Code)
			assert.Equal(t, "custom message", meta.Message)
			assert.Equal(t, []string{"firstName", "email"}, meta.Details)
		})

		t.Run("WithDetails は Details のみを付与する", func(t *testing.T) {
			t.Parallel()
			err := apperror.WithDetails(apperror.ErrValidation, "firstName")
			meta, ok := apperror.MetaFrom(err)
			require.True(t, ok)
			assert.Empty(t, meta.Code)
			assert.Empty(t, meta.Message)
			assert.Equal(t, []string{"firstName"}, meta.Details)
		})

		t.Run("さらにラップされていても抽出できる", func(t *testing.T) {
			t.Parallel()
			err := xerrors.Wrap(apperror.WithDetails(apperror.ErrValidation, "email"), "update failed")
			meta, ok := apperror.MetaFrom(err)
			require.True(t, ok)
			assert.Equal(t, []string{"email"}, meta.Details)
		})

		t.Run("多重に付与されている場合は外側が勝つ", func(t *testing.T) {
			t.Parallel()
			inner := apperror.WithMeta(apperror.ErrValidation, apperror.Meta{Code: "INNER"})
			outer := apperror.WithMeta(inner, apperror.Meta{Code: "OUTER"})
			meta, ok := apperror.MetaFrom(outer)
			require.True(t, ok)
			assert.Equal(t, "OUTER", meta.Code)
		})

		t.Run("抽出した Details を書き換えても元の Meta に影響しない", func(t *testing.T) {
			t.Parallel()
			err := apperror.WithDetails(apperror.ErrValidation, "firstName")
			meta, ok := apperror.MetaFrom(err)
			require.True(t, ok)
			meta.Details[0] = "mutated"
			again, ok := apperror.MetaFrom(err)
			require.True(t, ok)
			assert.Equal(t, []string{"firstName"}, again.Details)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("メタ無しエラーの場合 ok=false", func(t *testing.T) {
			t.Parallel()
			_, ok := apperror.MetaFrom(xerrors.Wrap(apperror.ErrValidation, "invalid"))
			assert.False(t, ok)
		})

		t.Run("nil の場合 ok=false", func(t *testing.T) {
			t.Parallel()
			_, ok := apperror.MetaFrom(nil)
			assert.False(t, ok)
		})
	})
}
