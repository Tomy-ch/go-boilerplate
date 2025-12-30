package security

import (
	"testing"

	"boilerplate-go/internal/config"

	"github.com/stretchr/testify/require"
)

func TestRateLimitModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, RateLimitModule())
}

func TestRateLimitMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	ipCfg := config.NewIPRateLimitConfig(cfg)

	out := RateLimitMiddleware(nil, ipCfg)
	require.Equal(t, rateLimitPriority, out.Middleware.Priority)
	require.NotNil(t, out.Middleware.Middleware)
}
