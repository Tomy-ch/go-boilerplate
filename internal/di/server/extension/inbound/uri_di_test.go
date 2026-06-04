package inbound

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreMiddleware(t *testing.T) {
	t.Parallel()

	mw := URIPreMiddleware()
	require.Equal(t, uriPrePriority, mw.Middleware.Priority)
	require.NotNil(t, mw.Middleware.Middleware)
}
