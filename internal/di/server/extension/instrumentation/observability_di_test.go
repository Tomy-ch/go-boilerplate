package instrumentation

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/require"
)

func TestObservabilityModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, ObservabilityModule())
}

func TestObservabilityMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	mw := ObservabilityMiddleware(appCfg)
	require.Equal(t, observabilityPriority, mw.Middleware.Priority)
	require.NotNil(t, mw.Middleware.Middleware)
}
