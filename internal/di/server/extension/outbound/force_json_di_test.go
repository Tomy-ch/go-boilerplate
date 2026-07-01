package outbound

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForceJSONMiddleware(t *testing.T) {
	t.Parallel()

	out := ForceJSONMiddleware()
	assert.Equal(t, forceJSONPriority, out.Middleware.Priority)
	assert.NotNil(t, out.Middleware.Middleware)
	assert.Equal(t, "forcejson", out.Middleware.Name)
}
