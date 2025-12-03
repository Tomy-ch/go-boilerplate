package cors

import (
	"testing"

	"boilerplate-go/internal/config"

	"github.com/stretchr/testify/require"
)

func TestModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, Module())
}

func TestUseMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	secCfg := config.NewSecurityConfig(cfg)

	out := UseMiddleware(secCfg)
	require.Equal(t, priority, out.Middleware.Priority)
	require.NotNil(t, out.Middleware.Middleware)
}
