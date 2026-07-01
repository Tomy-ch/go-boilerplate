package inbound

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_provideIPExtractorServeConfig(t *testing.T) {
	t.Parallel()

	_, parsedCIDR, err := net.ParseCIDR("192.168.0.0/16")
	require.NoError(t, err)

	newExtractor := func(t *testing.T, mode string) echo.IPExtractor {
		t.Helper()
		cfg := config.MockConfigForTest(t)
		appCfg := config.NewApplicationConfig(cfg)
		appCfg.SetApplicationMode(t, mode)
		secCfg := config.NewSecurityConfig(cfg)
		secCfg.SetCIDR(t, parsedCIDR)

		out := provideIPExtractorServeConfig(appCfg, secCfg)
		require.NotNil(t, out.SrvCfg)

		e := &echo.Echo{}
		out.SrvCfg(e)
		require.NotNil(t, e.IPExtractor)
		return e.IPExtractor
	}

	newReq := func() *http.Request {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		req.Header.Set(echo.HeaderXForwardedFor, "203.0.113.5")
		return req
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("本番モードでは信頼プロキシ経由のXFFからIPを抽出する", func(t *testing.T) {
			t.Parallel()
			extractor := newExtractor(t, config.ProductionMode)
			assert.Equal(t, "203.0.113.5", extractor(newReq()))
		})

		t.Run("開発モードでは接続元IPを直接抽出しXFFを無視する", func(t *testing.T) {
			t.Parallel()
			extractor := newExtractor(t, config.DevelopmentMode)
			assert.Equal(t, "192.168.1.1", extractor(newReq()))
		})
	})
}
