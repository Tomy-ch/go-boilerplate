package logging

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, Module())
}

func TestUseMiddleware(t *testing.T) {
	t.Parallel()

	out := UseMiddleware(zap.NewNop())
	require.Equal(t, priority, out.Middleware.Priority)
	require.NotNil(t, out.Middleware.Middleware)
}
