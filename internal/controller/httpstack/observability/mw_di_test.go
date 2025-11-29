package observability

import (
	"testing"

	"boilerplate-go/internal/config"

	"github.com/stretchr/testify/require"
)

func TestModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, Module())
}

func TestPrimitiveModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, PrimitiveModule())
}

func TestCoreModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, CoreModule())
}

func TestUseMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	mw := UseMiddleware(appCfg)
	require.Equal(t, priority, mw.Middleware.Priority)
	require.NotNil(t, mw.Middleware.Middleware)
}
