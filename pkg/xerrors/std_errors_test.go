package xerrors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewErrors(t *testing.T) {
	t.Parallel()

	e := NewErrors()
	require.NotNil(t, e)

	t.Run("New", func(t *testing.T) {
		t.Parallel()
		err := e.New("boom")
		require.EqualError(t, err, "boom")
	})

	t.Run("Wrap", func(t *testing.T) {
		t.Parallel()
		base := errors.New("base")
		err := e.Wrap(base, "ctx")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ctx")
		assert.True(t, e.Is(err, base))
	})

	t.Run("As", func(t *testing.T) {
		t.Parallel()
		var target *CustomError
		err := e.Wrap(&CustomError{}, "ctx")
		assert.True(t, e.As(err, &target))
	})

	t.Run("StackTrace", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, e.StackTrace(nil))
		assert.NotEmpty(t, e.StackTrace(e.New("x")))
	})
}
