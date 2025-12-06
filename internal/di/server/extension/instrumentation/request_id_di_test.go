package instrumentation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequestIDModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, RequestIDModule())
}

func TestRequestIDMiddleware(t *testing.T) {
	t.Parallel()

	mw := RequestIDMiddleware()
	require.Equal(t, requestIDPriority, mw.Middleware.Priority)
	require.NotNil(t, mw.Middleware.Middleware)
}
