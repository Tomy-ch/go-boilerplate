package outbound

import (
	"net/http"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubDetailPolicy は、DI 配線テスト用の DetailPolicy スタブです。
type stubDetailPolicy struct{}

func (stubDetailPolicy) Allows(*http.Request) bool { return false }

func Test_provideErrorHandlerServeConfig(t *testing.T) {
	t.Parallel()

	log := logging.NewTestLogger(t)
	obsCfg := config.NewObservabilityConfig(config.MockConfigForTest(t))
	lf := logging.NewTestLogFieldBuilder(t)

	out := provideErrorHandlerServeConfig(stubDetailPolicy{}, log, lf, obsCfg)
	require.NotNil(t, out.SrvCfg.Config)
	assert.Equal(t, "errorhandler", out.SrvCfg.Name)

	e := &echo.Echo{}
	out.SrvCfg.Config(e)
	assert.NotNil(t, e.HTTPErrorHandler)
}

func Test_provideDetailPolicy(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
