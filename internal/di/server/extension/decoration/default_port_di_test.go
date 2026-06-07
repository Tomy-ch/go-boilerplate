package decoration

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func Test_provideDefaultPortServeConfig(t *testing.T) {
	t.Parallel()

	appCfg := config.NewApplicationConfig(config.MockConfigForTest(t))
	out := provideDefaultPortServeConfig(appCfg)
	require.NotNil(t, out)
	require.NotNil(t, out.SrvCfg)

	e := &echo.Echo{}
	out.SrvCfg(e)
	require.NotNil(t, e.HidePort)
}
