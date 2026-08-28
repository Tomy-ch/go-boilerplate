package instrumentation

import (
	"testing"

	"go-boilerplate/internal/controller/httpstack/redaction"
	"go-boilerplate/internal/logging"

	"github.com/stretchr/testify/assert"
)

func TestLoggingMiddleware(t *testing.T) {
	t.Parallel()

	lf := logging.NewTestLogFieldBuilder(t)

	out := LoggingMiddleware(logging.NewTestLogger(t), lf, redaction.Redactor{})
	assert.Equal(t, loggingPriority, out.Middleware.Priority)
	assert.NotNil(t, out.Middleware.Middleware)
}
