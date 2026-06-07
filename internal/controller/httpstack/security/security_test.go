package security

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSecureConfig(t *testing.T) {
	t.Parallel()
	cfg := config.MockConfigForTest(t)
	secCfg := config.NewSecurityConfig(cfg)

	scfg := buildSecureConfig(secCfg)

	assert.Equal(t, secCfg.ContentTypeNosniff(), scfg.ContentTypeNosniff)
	assert.Equal(t, secCfg.ReferrerPolicy(), scfg.ReferrerPolicy)
	assert.Equal(t, secCfg.XFrameOptions(), scfg.XFrameOptions)
	assert.Equal(t, int(secCfg.HSTSMaxAge().Seconds()), scfg.HSTSMaxAge)
	assert.Equal(t, secCfg.HSTSExcludeSubdomains(), scfg.HSTSExcludeSubdomains)
	assert.Equal(t, secCfg.HSTSPreloadEnabled(), scfg.HSTSPreloadEnabled)
}

func TestMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	secCfg := config.NewSecurityConfig(cfg)

	require.NotNil(t, Middleware(secCfg))
}
