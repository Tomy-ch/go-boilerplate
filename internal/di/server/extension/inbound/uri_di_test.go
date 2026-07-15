package inbound

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURIPreMiddleware(t *testing.T) {
	t.Parallel()

	mw := URIPreMiddleware()
	assert.Equal(t, uriPrePriority, mw.Middleware.Priority)
	assert.NotNil(t, mw.Middleware.Middleware)
}
