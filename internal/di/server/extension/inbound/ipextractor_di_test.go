package inbound

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestIPExtractorModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, IPExtractorModule())
}

func Test_provideIPExtractorServeConfig(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	secCfg := config.NewSecurityConfig(cfg)

	out := provideIPExtractorServeConfig(appCfg, secCfg)
	require.NotNil(t, out)
	require.NotNil(t, out.SrvCfg)

	e := &echo.Echo{}
	out.SrvCfg(e)
	require.NotNil(t, e.IPExtractor)
}
