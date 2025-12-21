package usecasetest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExpectedDBError(t *testing.T) {
	t.Parallel()

	actual := ExpectedDBError(t)
	require.Error(t, actual)
}

func TestMockTransactionManager(t *testing.T) {
	t.Parallel()

	actual := NewMockTransactionManager(t)
	require.NotNil(t, actual)
}
