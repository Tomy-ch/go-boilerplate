package server

import (
	"net"
	"testing"

	"boilerplate-go/internal/config"

	"github.com/stretchr/testify/require"
)

func TestNewIPExtractor(t *testing.T) {
	cidr := "192.168.0.0/16"
	_, parsedCIDR, err := net.ParseCIDR(cidr)
	require.NoError(t, err)

	t.Run("本番モードの場合", func(t *testing.T) {
		require.NoError(t, config.Load())
		t.Setenv("SERVER_APP_MODE", config.ProductionMode)
		t.Setenv("SECURITY_CIDR", cidr)
		cfg, err := config.New()
		require.NoError(t, err)
		require.Equal(t, config.ProductionMode, cfg.ServerAppMode())
		require.Equal(t, parsedCIDR.String(), cfg.CIDR().String())

		actual := NewIPExtractor(cfg)
		require.NotNil(t, actual)
	})

	t.Run("開発モードの場合", func(t *testing.T) {
		require.NoError(t, config.Load())
		t.Setenv("APP_MODE", config.DevelopmentMode)
		t.Setenv("SECURITY_CIDR", cidr)
		cfg, err := config.New()
		require.NoError(t, err)
		require.Equal(t, config.DevelopmentMode, cfg.ServerAppMode())
		require.Equal(t, parsedCIDR.String(), cfg.CIDR().String())

		actual := NewIPExtractor(cfg)
		require.NotNil(t, actual)
	})

	t.Run("開発モードがない場合", func(t *testing.T) {
		extractor := NewIPExtractor(&config.Config{})
		require.NotNil(t, extractor)
	})
}
