package recovery

import (
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/logging"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, Module())
}

func TestUseMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	lf := logging.NewLogFields(config.NewObservabilityConfig(cfg))
	z := zap.NewNop()

	mw := UseMiddleware(z, lf, appCfg)

	require.Equal(t, priority, mw.Middleware.Priority)
	require.NotNil(t, mw.Middleware.Middleware)
}
