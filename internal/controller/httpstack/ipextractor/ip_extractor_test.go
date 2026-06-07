package ipextractor

import (
	"net"
	"testing"

	"go-boilerplate/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	e := &echo.Echo{}
	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	secCfg := config.NewSecurityConfig(cfg)

	New(e, appCfg, secCfg)
	require.NotNil(t, e.IPExtractor)
}

func TestNewIPExtractor(t *testing.T) {
	t.Parallel()

	cidr := "192.168.0.0/16"
	_, parsedCIDR, err := net.ParseCIDR(cidr)
	require.NoError(t, err)

	t.Run("本番モードの場合", func(t *testing.T) {
		t.Parallel()

		cfg := config.MockConfigForTest(t)

		appCfg := config.NewApplicationConfig(cfg)
		appCfg.SetApplicationMode(t, config.ProductionMode)

		secCfg := config.NewSecurityConfig(cfg)
		secCfg.SetCIDR(t, parsedCIDR)

		assert.Equal(t, config.ProductionMode, appCfg.Mode())
		assert.Equal(t, parsedCIDR.String(), secCfg.CIDR().String())
		actual := NewIPExtractor(appCfg, secCfg)
		require.NotNil(t, actual)
	})

	t.Run("開発モードの場合", func(t *testing.T) {
		t.Parallel()

		cfg := config.MockConfigForTest(t)
		appCfg := config.NewApplicationConfig(cfg)
		appCfg.SetApplicationMode(t, config.DevelopmentMode)

		secCfg := config.NewSecurityConfig(cfg)
		secCfg.SetCIDR(t, parsedCIDR)

		assert.Equal(t, config.DevelopmentMode, appCfg.Mode())
		assert.Equal(t, parsedCIDR.String(), secCfg.CIDR().String())
		actual := NewIPExtractor(appCfg, secCfg)
		require.NotNil(t, actual)
	})

	t.Run("開発モードがない場合", func(t *testing.T) {
		extractor := NewIPExtractor(&config.ApplicationConfig{}, &config.SecurityConfig{})
		require.NotNil(t, extractor)
	})
}
