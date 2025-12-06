package outbound

import (
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/logging"

	"github.com/stretchr/testify/require"
)

func TestRecoveryModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, RecoveryModule())
}

func TestRecoveryMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	lf := logging.NewLogFields(config.NewObservabilityConfig(cfg))
	logger := logging.NewTestInstance(t)

	mw := RecoveryMiddleware(logger, lf, appCfg)

	require.Equal(t, recoveryPriority, mw.Middleware.Priority)
	require.NotNil(t, mw.Middleware.Middleware)
}
