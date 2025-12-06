package outbound

import (
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/logging"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestErrorHandlerModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, ErrorHandlerModule())
}

func Test_provideErrorHandlerServeConfig(t *testing.T) {
	t.Parallel()

	log := logging.NewTestInstance(t)
	obsCfg := config.NewObservabilityConfig(config.MockConfigForTest(t))
	lf := logging.NewLogFields(obsCfg)

	out := provideErrorHandlerServeConfig(log, lf, obsCfg)
	require.NotNil(t, out)
	require.NotNil(t, out.SrvCfg)

	e := &echo.Echo{}
	out.SrvCfg(e)
	require.NotNil(t, e.HTTPErrorHandler)
}
