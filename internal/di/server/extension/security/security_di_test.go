package security

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/require"
)

func TestModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, Module())
}

func TestMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	secCfg := config.NewSecurityConfig(cfg)

	mw := Middleware(secCfg)
	require.Equal(t, securityPriority, mw.Middleware.Priority)
	require.NotNil(t, mw.Middleware.Middleware)
}
