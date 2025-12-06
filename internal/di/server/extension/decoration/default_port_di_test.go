package decoration

import (
	"testing"

	"boilerplate-go/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestDefaultPortModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, DefaultPortModule())
}

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
