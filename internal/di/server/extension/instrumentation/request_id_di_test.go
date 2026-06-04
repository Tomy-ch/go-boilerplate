package instrumentation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestIDMiddleware(t *testing.T) {
	t.Parallel()

	mw := RequestIDMiddleware()
	assert.Equal(t, requestIDPriority, mw.Middleware.Priority)
	require.NotNil(t, mw.Middleware.Middleware)
}
