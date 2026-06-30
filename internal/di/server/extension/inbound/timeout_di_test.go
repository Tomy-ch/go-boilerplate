package inbound

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go-boilerplate/internal/config"
)

func TestTimeoutPreMiddleware(t *testing.T) {
	t.Parallel()

	srvCfg := config.NewServerConfig(config.MockConfigForTest(t))
	mw := TimeoutPreMiddleware(srvCfg)

	assert.Equal(t, "timeout", mw.Middleware.Name)
	assert.Equal(t, timeoutPrePriority, mw.Middleware.Priority)
	assert.NotNil(t, mw.Middleware.Middleware)
	// deadline budget の元となる RequestTimeout が正の値であることを確認する。
	assert.Positive(t, srvCfg.RequestTimeout())
}
