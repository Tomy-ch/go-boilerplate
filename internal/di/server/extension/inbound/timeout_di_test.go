package inbound

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/config"
)

func TestTimeoutPreMiddleware(t *testing.T) {
	t.Parallel()

	srvCfg := config.NewServerConfig(config.MockConfigForTest(t))
	mw := TimeoutPreMiddleware(srvCfg)

	assert.Equal(t, "timeout", mw.Middleware.Name)
	assert.Equal(t, timeoutPrePriority, mw.Middleware.Priority)
	require.NotNil(t, mw.Middleware.Middleware)
}
