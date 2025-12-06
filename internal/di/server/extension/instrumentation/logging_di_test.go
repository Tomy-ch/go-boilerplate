package instrumentation

import (
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/logging"

	"github.com/stretchr/testify/require"
)

func TestLoggingModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, LoggingModule())
}

func TestLoggingMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewLogFields(obsCfg)

	out := LoggingMiddleware(logging.NewTestInstance(t), lf)
	require.Equal(t, loggingPriority, out.Middleware.Priority)
	require.NotNil(t, out.Middleware.Middleware)
}
