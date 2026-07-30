package testkit

import (
	"context"
	"testing"

	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpectedDBError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非nilのエラーを返す", func(t *testing.T) {
			t.Parallel()
			actual := ExpectedDBError()
			require.Error(t, actual)
		})

		t.Run("呼び出しごとに同一のエラーを返しerrors.Isで識別できる", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, ExpectedDBError(), errExpectedDB)
		})
	})
}

func TestNewMockTransactionManager(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Do がコールバックを実行する", func(t *testing.T) {
			t.Parallel()

			mgr := NewMockTransactionManager(t)
			called := false
			err := mgr.Do(context.Background(), func(context.Context) error {
				called = true
				return nil
			})
			require.NoError(t, err)
			assert.True(t, called)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Do がコールバックの error を透過する", func(t *testing.T) {
			t.Parallel()

			mgr := NewMockTransactionManager(t)
			want := xerrors.New("boom")
			err := mgr.Do(context.Background(), func(context.Context) error { return want })
			require.ErrorIs(t, err, want)
		})
	})
}
