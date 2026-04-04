package integration

import (
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/handler/version"
	"go-boilerplate/internal/controller/handler/version/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/system"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestVersionIntegration(t *testing.T) {
	t.Parallel()

	t.Run("GET /versionのエンドポイントが正常に動作することを確認する", func(t *testing.T) {
		e := echo.New()
		tf := observability.NewNoopTracerFactory(t)

		bi := system.NewBuildInfo()
		cfg := config.MockConfigForTest(t)
		appCfg := config.NewApplicationConfig(cfg)
		osCfg := config.NewOperationSystemConfig(cfg)

		loc, err := time.LoadLocation(osCfg.TimeZone())
		require.NoError(t, err)

		version.BindHandler(e, tf, loc, bi, appCfg)
		actual := StartServer(t, e).DoJSON(http.MethodGet, "/version", nil, nil)
		AssertJSONResponse(t, gen.VersionResponse{}, actual)
	})
}
