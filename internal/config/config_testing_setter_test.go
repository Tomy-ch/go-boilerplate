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
		cfg.app.SetServerAppMode(t, expected)
		require.Equal(t, expected, cfg.app.AppMode())
	})

	t.Run("SetDatabaseDriver", func(t *testing.T) {
		expected := "test-driver"
		cfg.database.SetDatabaseDriver(t, expected)
		require.Equal(t, expected, cfg.database.DatabaseDriver())
	})

	t.Run("SetDatabaseHost", func(t *testing.T) {
		expected := "test-host"
		cfg.database.SetDatabaseHost(t, expected)
		require.Equal(t, expected, cfg.database.DatabaseHost())
	})

	t.Run("SetDatabaseName", func(t *testing.T) {
		expected := "test-name"
		cfg.database.SetDatabaseName(t, expected)
		require.Equal(t, expected, cfg.database.DatabaseName())
	})

	t.Run("SetCIDR", func(t *testing.T) {
		_, testCIDR, _ := net.ParseCIDR("192.168.1.0/24")
		cfg.security.SetCIDR(t, testCIDR)
		require.Equal(t, testCIDR, cfg.security.CIDR())
	})
}
