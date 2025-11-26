package banner

import (
	"testing"

	"boilerplate-go/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, Module())
}

func Test_provideServeConfig(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	out := provideServeConfig(cfg)
	require.NotNil(t, out)
	require.NotNil(t, out.SrvCfg)

	e := &echo.Echo{}
	out.SrvCfg(e)
	require.NotNil(t, e.HidePort)
}
