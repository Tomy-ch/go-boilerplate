package security

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/require"
)

func TestCORSModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, CORSModule())
}

func TestCORSMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	secCfg := config.NewSecurityConfig(cfg)

	out := CORSMiddleware(secCfg)
	require.Equal(t, corsPriority, out.Middleware.Priority)
	require.NotNil(t, out.Middleware.Middleware)
}
