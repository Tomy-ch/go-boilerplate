package instrumentation

import (
	"testing"

	"go-boilerplate/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggingMiddleware(t *testing.T) {
	t.Parallel()

	lf := logging.NewTestLogFieldBuilder(t)

	out := LoggingMiddleware(logging.NewTestLogger(t), lf)
	assert.Equal(t, loggingPriority, out.Middleware.Priority)
	require.NotNil(t, out.Middleware.Middleware)
}
