package recovery

import (
	"testing"

	"boilerplate-go/internal/config"

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
	z := zap.NewNop()

	mw := UseMiddleware(z, appCfg)

	require.Equal(t, priority, mw.Middleware.Priority)
	require.NotNil(t, mw.Middleware.Middleware)
}
