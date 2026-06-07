package xerrors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type CustomError struct{}

func (e *CustomError) Error() string {
	return "custom error"
}

func TestNew(t *testing.T) {
	t.Parallel()
	errStr := "test error"
	err := New(errStr)
	require.EqualError(t, err, errStr)
}

func TestWrap(t *testing.T) {
	t.Parallel()
	warpStr := "wrapped error"
	baseErr := errors.New("base error")
	actual := Wrap(baseErr, warpStr)
	require.Error(t, actual)
	assert.Contains(t, actual.Error(), warpStr)
	assert.Contains(t, actual.Error(), baseErr.Error())
}

func TestIs(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("base error")
	t.Run("Is が true を返すケース", func(t *testing.T) {
		t.Parallel()
		wrappedErr := Wrap(baseErr, "wrapped error")
		assert.True(t, Is(wrappedErr, baseErr))
	})

	t.Run("Is が false を返すケース", func(t *testing.T) {
		t.Parallel()
		anotherErr := errors.New("another error")
		assert.False(t, Is(baseErr, anotherErr))
	})
}

func TestAs(t *testing.T) {
	t.Parallel()
	targetErr := &CustomError{}
	wrappedErr := Wrap(targetErr, "wrapped error")

	var extractedErr *CustomError

	t.Run("As が true を返すケース", func(t *testing.T) {
		t.Parallel()

		assert.True(t, As(wrappedErr, &extractedErr))
		assert.Equal(t, targetErr, extractedErr)
	})

	t.Run("As が false を返すケース", func(t *testing.T) {
		t.Parallel()
		err := errors.New("not-custom")
		assert.False(t, As(err, &extractedErr))
	})
}

func TestStackTrace(t *testing.T) {
	t.Parallel()

	t.Run("nil の場合は空文字を返す", func(t *testing.T) {
		t.Parallel()
		require.Empty(t, StackTrace(nil))
	})

	t.Run("エラーがある場合はメッセージを含むスタック文字列を返す", func(t *testing.T) {
		t.Parallel()
		err := New("stack-msg")
		st := StackTrace(err)
		require.NotEmpty(t, st)
		assert.Contains(t, st, "stack-msg")
	})
}
