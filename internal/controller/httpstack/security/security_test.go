package security

import (
	"testing"

	"boilerplate-go/internal/config"

	"github.com/stretchr/testify/require"
)

func TestBuildSecureConfig(t *testing.T) {
	t.Parallel()
	cfg := config.MockConfigForTest(t)
	secCfg := config.NewSecurityConfig(cfg)

	scfg := buildSecureConfig(secCfg)

	require.Equal(t, secCfg.ContentTypeNosniff(), scfg.ContentTypeNosniff)
	require.Equal(t, secCfg.ReferrerPolicy(), scfg.ReferrerPolicy)
	require.Equal(t, secCfg.XFrameOptions(), scfg.XFrameOptions)
	require.Equal(t, int(secCfg.HSTSMaxAge().Seconds()), scfg.HSTSMaxAge)
	require.Equal(t, secCfg.HSTSExcludeSubdomains(), scfg.HSTSExcludeSubdomains)
	require.Equal(t, secCfg.HSTSPreloadEnabled(), scfg.HSTSPreloadEnabled)
}

func TestMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	secCfg := config.NewSecurityConfig(cfg)

	require.NotNil(t, Middleware(secCfg))
}
