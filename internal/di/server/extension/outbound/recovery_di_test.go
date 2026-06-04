package outbound

import (
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"

	"github.com/stretchr/testify/require"
)

func TestRecoveryMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(t)
	logger := logging.NewTestLogger(t)

	mw := RecoveryMiddleware(logger, lf, appCfg)

	require.Equal(t, recoveryPriority, mw.Middleware.Priority)
	require.NotNil(t, mw.Middleware.Middleware)
}
