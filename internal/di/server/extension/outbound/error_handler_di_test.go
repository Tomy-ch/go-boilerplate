package outbound

import (
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_provideErrorHandlerServeConfig(t *testing.T) {
	t.Parallel()

	log := logging.NewTestLogger(t)
	obsCfg := config.NewObservabilityConfig(config.MockConfigForTest(t))
	lf := logging.NewTestLogFieldBuilder(t)

	out := provideErrorHandlerServeConfig(log, lf, obsCfg)
	require.NotNil(t, out.SrvCfg.Config)
	assert.Equal(t, "errorhandler", out.SrvCfg.Name)

	e := &echo.Echo{}
	out.SrvCfg.Config(e)
	assert.NotNil(t, e.HTTPErrorHandler)
}
