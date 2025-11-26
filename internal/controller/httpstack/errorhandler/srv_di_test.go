package errorhandler

import (
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestModule(t *testing.T) {
	t.Parallel()
	require.NotNil(t, Module())
}

func Test_provideServeConfig(t *testing.T) {
	t.Parallel()

	z := zap.NewNop()
	out := provideServeConfig(z)
	require.NotNil(t, out)
	require.NotNil(t, out.SrvCfg)

	e := &echo.Echo{}
	out.SrvCfg(e)
	require.NotNil(t, e.HTTPErrorHandler)
}
