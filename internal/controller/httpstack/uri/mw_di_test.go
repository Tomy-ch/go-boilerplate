package uri

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, Module())
}

func TestPreMiddleware(t *testing.T) {
	t.Parallel()

	mw := PreMiddleware()
	require.Equal(t, priority, mw.Middleware.Priority)
	require.NotNil(t, mw.Middleware.Middleware)
}
