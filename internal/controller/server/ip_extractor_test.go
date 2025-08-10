package server

import (
	"net"
	"testing"

	"boilerplate-go/internal/appconfig"
	"boilerplate-go/internal/env"

	"github.com/stretchr/testify/require"
)

func TestNewIPExtractor(t *testing.T) {
	cidr := "192.168.0.0/16"
	_, parsedCIDR, err := net.ParseCIDR(cidr)
	require.NoError(t, err)

	t.Run("本番モードの場合", func(t *testing.T) {
		require.NoError(t, env.Load())
		t.Setenv("APP_MODE", appconfig.ProductionMode)
		t.Setenv("SECURITY_CIDR", cidr)
		cfg, err := appconfig.New()
		require.NoError(t, err)
		require.Equal(t, appconfig.ProductionMode, cfg.AppMode())
		require.Equal(t, parsedCIDR.String(), cfg.CIDR().String())

		actual := NewIPExtractor(cfg)
		require.NotNil(t, actual)
	})

	t.Run("開発モードの場合", func(t *testing.T) {
		require.NoError(t, env.Load())
		t.Setenv("APP_MODE", appconfig.DevelopmentMode)
		t.Setenv("SECURITY_CIDR", cidr)
		cfg, err := appconfig.New()
		require.NoError(t, err)
		require.Equal(t, appconfig.DevelopmentMode, cfg.AppMode())
		require.Equal(t, parsedCIDR.String(), cfg.CIDR().String())

		actual := NewIPExtractor(cfg)
		require.NotNil(t, actual)
	})

	t.Run("開発モードがない場合", func(t *testing.T) {
		extractor := NewIPExtractor(&appconfig.Config{})
		require.NotNil(t, extractor)
	})
}
