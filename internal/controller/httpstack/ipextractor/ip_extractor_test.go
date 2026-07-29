package ipextractor

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Echoに非nilのIPExtractorが設定される", func(t *testing.T) {
			t.Parallel()
			e := &echo.Echo{}
			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			secCfg := config.NewSecurityConfig(cfg)

			New(e, appCfg, secCfg)
			assert.NotNil(t, e.IPExtractor)
		})
	})
}

func TestNewIPExtractor(t *testing.T) {
	t.Parallel()

	cidr := "192.168.0.0/16"
	_, parsedCIDR, err := net.ParseCIDR(cidr)
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("本番モードでは信頼プロキシ経由のXFFヘッダからIPを抽出する", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			appCfg.SetApplicationMode(t, config.ProductionMode)

			secCfg := config.NewSecurityConfig(cfg)
			secCfg.SetCIDR(t, parsedCIDR)

			actual := NewIPExtractor(appCfg, secCfg)
			require.NotNil(t, actual)

			// 接続元(RemoteAddr)を信頼 CIDR(192.168.0.0/16)内に置くと、XFF のクライアント IP が採用される。
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			req.RemoteAddr = "192.168.1.1:1234"
			req.Header.Set(echo.HeaderXForwardedFor, "203.0.113.5")
			assert.Equal(t, "203.0.113.5", actual(req))
		})

		t.Run("開発モードでは接続元IPを直接抽出しXFFを無視する", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			appCfg := config.NewApplicationConfig(cfg)
			appCfg.SetApplicationMode(t, config.DevelopmentMode)

			secCfg := config.NewSecurityConfig(cfg)
			secCfg.SetCIDR(t, parsedCIDR)

			actual := NewIPExtractor(appCfg, secCfg)
			require.NotNil(t, actual)

			// direct 抽出は RemoteAddr をそのまま採用し、XFF は無視する。
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			req.RemoteAddr = "192.168.1.1:1234"
			req.Header.Set(echo.HeaderXForwardedFor, "203.0.113.5")
			assert.Equal(t, "192.168.1.1", actual(req))
		})

		t.Run("モード未設定でも非nilのextractorを返す", func(t *testing.T) {
			t.Parallel()
			extractor := NewIPExtractor(&config.ApplicationConfig{}, &config.SecurityConfig{})
			assert.NotNil(t, extractor)
		})
	})
}
