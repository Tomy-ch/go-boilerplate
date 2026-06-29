package inbound

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go-boilerplate/internal/config"
)

func TestBodyLimitPreMiddleware(t *testing.T) {
	t.Parallel()

	srvCfg := config.NewServerConfig(config.MockConfigForTest(t))
	mw := BodyLimitPreMiddleware(srvCfg)

	assert.Equal(t, "bodyLimit", mw.Middleware.Name)
	assert.Equal(t, bodyLimitPrePriority, mw.Middleware.Priority)
	assert.NotNil(t, mw.Middleware.Middleware)
}
