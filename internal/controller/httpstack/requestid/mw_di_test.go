package requestid

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

	mw := UseMiddleware()
	require.Equal(t, priority, mw.Middleware.Priority)
	require.NotNil(t, mw.Middleware.Middleware)
}
