package logging

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewTestInstance(t *testing.T) {
	t.Parallel()

	expected := &logger{log: zap.NewNop()}
	actual := NewTestInstance(t)

	require.Equal(t, expected, actual)
}
