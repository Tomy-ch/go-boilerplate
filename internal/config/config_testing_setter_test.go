package config

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigTestingSetters(t *testing.T) {
	cfg := MockConfigForTest(t)

	t.Run("SetAppMode", func(t *testing.T) {
		expected := "test-mode"
		cfg.SetServerAppMode(t, expected)
		require.Equal(t, expected, cfg.AppMode())
	})

	t.Run("SetDatabaseDriver", func(t *testing.T) {
		expected := "test-driver"
		cfg.SetDatabaseDriver(t, expected)
		require.Equal(t, expected, cfg.DatabaseDriver())
	})

	t.Run("SetDatabaseHost", func(t *testing.T) {
		expected := "test-host"
		cfg.SetDatabaseHost(t, expected)
		require.Equal(t, expected, cfg.DatabaseHost())
	})

	t.Run("SetDatabaseName", func(t *testing.T) {
		expected := "test-name"
		cfg.SetDatabaseName(t, expected)
		require.Equal(t, expected, cfg.DatabaseName())
	})

	t.Run("SetCIDR", func(t *testing.T) {
		_, testCIDR, _ := net.ParseCIDR("192.168.1.0/24")
		cfg.SetCIDR(t, testCIDR)
		require.Equal(t, testCIDR, cfg.CIDR())
	})
}
