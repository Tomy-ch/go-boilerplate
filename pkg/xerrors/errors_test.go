package xerrors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type CustomError struct{}

func (e *CustomError) Error() string {
	return "custom error"
}

func TestCockroachDBError_New(t *testing.T) {
	t.Parallel()
	errStr := "test error"
	err := New(errStr)
	require.EqualError(t, err, errStr)
}

func TestCockroachDBError_Wrap(t *testing.T) {
	t.Parallel()
	warpStr := "wrapped error"
	baseErr := errors.New("base error")
	actual := Wrap(baseErr, warpStr)
	require.Error(t, actual)
	require.Contains(t, actual.Error(), warpStr)
	require.Contains(t, actual.Error(), baseErr.Error())
}

func TestCockroachDBError_Is(t *testing.T) {
	t.Parallel()
	baseErr := errors.New("base error")
	wrappedErr := Wrap(baseErr, "wrapped error")
	require.True(t, Is(wrappedErr, baseErr))
}

func TestCockroachDBError_As(t *testing.T) {
	t.Parallel()
	targetErr := &CustomError{}
	wrappedErr := Wrap(targetErr, "wrapped error")

	var extractedErr *CustomError
	require.True(t, As(wrappedErr, &extractedErr))
	require.Equal(t, targetErr, extractedErr)
}
