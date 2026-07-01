package instrumentation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestIDMiddleware(t *testing.T) {
	t.Parallel()

	mw := RequestIDMiddleware()
	assert.Equal(t, requestIDPriority, mw.Middleware.Priority)
	assert.NotNil(t, mw.Middleware.Middleware)
}
