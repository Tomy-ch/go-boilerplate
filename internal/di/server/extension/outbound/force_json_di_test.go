package outbound

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestForceJSONModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, ForceJSONModule())
}

func TestForceJSONMiddleware(t *testing.T) {
	t.Parallel()

	out := ForceJSONMiddleware()
	require.Equal(t, forceJSONPriority, out.Middleware.Priority)
	require.NotNil(t, out.Middleware.Middleware)
}
