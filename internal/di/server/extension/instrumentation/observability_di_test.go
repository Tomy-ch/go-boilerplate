package instrumentation

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObservabilityMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	mw := ObservabilityMiddleware(appCfg)
	assert.Equal(t, observabilityPriority, mw.Middleware.Priority)
	require.NotNil(t, mw.Middleware.Middleware)
}
