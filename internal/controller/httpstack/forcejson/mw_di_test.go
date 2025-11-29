package forcejson

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, Module())
}

func TestUseMiddleware(t *testing.T) {
	t.Parallel()

	out := UseMiddleware()
	require.Equal(t, priority, out.Middleware.Priority)
	require.NotNil(t, out.Middleware.Middleware)
}
