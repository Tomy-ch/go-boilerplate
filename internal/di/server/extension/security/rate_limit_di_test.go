package security

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/require"
)

func TestRateLimitMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	ipCfg := config.NewIPRateLimitConfig(cfg)

	out := RateLimitMiddleware(nil, ipCfg)
	require.Equal(t, rateLimitPriority, out.Middleware.Priority)
	require.NotNil(t, out.Middleware.Middleware)
}
