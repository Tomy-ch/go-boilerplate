package server

import (
	"testing"

	"boilerplate-go/internal/config"

	"github.com/stretchr/testify/require"
)

func TestNewAppServer(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	srvCfg := config.NewServerConfig(cfg)

	actual := NewAppServer(srvCfg)
	require.Equal(t, srvCfg.ReadHeaderTimeout(), actual.Server.ReadHeaderTimeout)
	require.Equal(t, srvCfg.ReadTimeout(), actual.Server.ReadTimeout)
	require.Equal(t, srvCfg.WriteTimeout(), actual.Server.WriteTimeout)
	require.Equal(t, srvCfg.IdleTimeout(), actual.Server.IdleTimeout)
	require.NotNil(t, actual)
}
