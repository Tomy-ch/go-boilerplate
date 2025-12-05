package logging

import (
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/logging"

	"github.com/stretchr/testify/require"
)

func TestModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, Module())
}

func TestUseMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewLogFields(obsCfg)

	out := UseMiddleware(logging.NewTestInstance(t), lf)
	require.Equal(t, priority, out.Middleware.Priority)
	require.NotNil(t, out.Middleware.Middleware)
}
