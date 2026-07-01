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
	})
}

func TestMockTransactionManager(t *testing.T) {
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

	t.Run("Do がコールバックの error を透過する", func(t *testing.T) {
		t.Parallel()

		mgr := NewMockTransactionManager(t)
		want := xerrors.New("boom")
		err := mgr.Do(context.Background(), func(context.Context) error { return want })
		require.ErrorIs(t, err, want)
	})
}
