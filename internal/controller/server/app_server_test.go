package server

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAppServer(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	srvCfg := config.NewServerConfig(cfg)

	actual := NewAppServer(srvCfg)
	assert.Equal(t, srvCfg.ReadHeaderTimeout(), actual.Server.ReadHeaderTimeout)
	assert.Equal(t, srvCfg.ReadTimeout(), actual.Server.ReadTimeout)
	assert.Equal(t, srvCfg.WriteTimeout(), actual.Server.WriteTimeout)
	assert.Equal(t, srvCfg.IdleTimeout(), actual.Server.IdleTimeout)
	require.NotNil(t, actual)
}
