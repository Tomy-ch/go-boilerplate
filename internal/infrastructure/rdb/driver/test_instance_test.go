package driver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTestInstance(t *testing.T) {
	t.Parallel()

	actual := NewTestInstance(t)
	require.NotNil(t, actual)
	require.IsType(t, &dbDriver{}, actual)
}
