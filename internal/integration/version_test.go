package integration

import (
	"net/http"
	"testing"
	"time"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/handler/handlertest/testinstance"
	"boilerplate-go/internal/controller/handler/version"
	"boilerplate-go/internal/controller/handler/version/gen"
	"boilerplate-go/internal/system"

	"github.com/stretchr/testify/require"
)

func TestVersionIntegration(t *testing.T) {
	t.Parallel()

	t.Run("GET /versionのエンドポイントが正常に動作することを確認する", func(t *testing.T) {
		e, _, tf, _ := testinstance.NewTestInstanceForBindHandler(t)

		bi := system.NewBuildInfo()
		cfg := config.MockConfigForTest(t)
		appCfg := config.NewApplicationConfig(cfg)
		osCfg := config.NewOSConfig(cfg)

		loc, err := time.LoadLocation(osCfg.TimeZone())
		require.NoError(t, err)

		version.BindHandler(e, tf, loc, bi, appCfg)
		actual := StartServer(t, e).DoJSON(http.MethodGet, "/version", nil, nil)
		AssertJSONResponse(t, gen.ResponseVersion{}, actual)
	})
}
