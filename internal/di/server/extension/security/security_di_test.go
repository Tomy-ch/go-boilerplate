package security

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	secCfg := config.NewSecurityConfig(cfg)

	mw := Middleware(secCfg)
	assert.Equal(t, securityPriority, mw.Middleware.Priority)
	require.NotNil(t, mw.Middleware.Middleware)
}
