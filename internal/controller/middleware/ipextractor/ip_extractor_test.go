package ipextractor

import (
	"net"
	"testing"

	"boilerplate-go/internal/config"

	"github.com/stretchr/testify/require"
)

func TestNewIPExtractor(t *testing.T) {
	t.Parallel()

	cidr := "192.168.0.0/16"
	_, parsedCIDR, err := net.ParseCIDR(cidr)
	require.NoError(t, err)

	t.Run("本番モードの場合", func(t *testing.T) {
		t.Parallel()

		cfg := config.MockConfigForTest(t)
		cfg.SetServerAppMode(t, config.ProductionMode)
		cfg.SetCIDR(t, parsedCIDR)
		require.Equal(t, config.ProductionMode, cfg.ServerAppMode())
		require.Equal(t, parsedCIDR.String(), cfg.CIDR().String())

		actual := NewIPExtractor(&cfg)
		require.NotNil(t, actual)
	})

	t.Run("開発モードの場合", func(t *testing.T) {
		t.Parallel()

		cfg := config.MockConfigForTest(t)
		cfg.SetServerAppMode(t, config.DevelopmentMode)
		cfg.SetCIDR(t, parsedCIDR)
		require.Equal(t, config.DevelopmentMode, cfg.ServerAppMode())
		require.Equal(t, parsedCIDR.String(), cfg.CIDR().String())

		actual := NewIPExtractor(&cfg)
		require.NotNil(t, actual)
	})

	t.Run("開発モードがない場合", func(t *testing.T) {
		extractor := NewIPExtractor(&config.Config{})
		require.NotNil(t, extractor)
	})
}
