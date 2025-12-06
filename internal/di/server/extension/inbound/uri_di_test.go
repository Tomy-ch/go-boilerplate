package inbound

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestURIModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, URIModule())
}

func TestPreMiddleware(t *testing.T) {
	t.Parallel()

	mw := URIPreMiddleware()
	require.Equal(t, uriPrePriority, mw.Middleware.Priority)
	require.NotNil(t, mw.Middleware.Middleware)
}
