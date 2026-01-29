package config

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfigTestingSetters(t *testing.T) {
	cfg := MockConfigForTest(t)

	t.Run("SetAppMode", func(t *testing.T) {
		expected := "test-mode"
		cfg.app.SetServerAppMode(t, expected)
		require.Equal(t, expected, cfg.app.Mode())
	})

	t.Run("SetAppEnv", func(t *testing.T) {
		expected := "test-env"
		cfg.app.SetAppEnv(t, expected)
		require.Equal(t, expected, cfg.app.Env())
	})

	t.Run("SetDatabaseDriver", func(t *testing.T) {
		expected := "test-driver"
		cfg.database.SetDatabaseDriver(t, expected)
		require.Equal(t, expected, cfg.database.Driver())
	})

	t.Run("SetDatabaseHost", func(t *testing.T) {
		expected := "test-host"
		cfg.database.SetDatabaseHost(t, expected)
		require.Equal(t, expected, cfg.database.Host())
	})

	t.Run("SetDatabaseName", func(t *testing.T) {
		expected := "test-name"
		cfg.database.SetDatabaseName(t, expected)
		require.Equal(t, expected, cfg.database.DBName())
	})

	t.Run("SetCIDR", func(t *testing.T) {
		_, testCIDR, _ := net.ParseCIDR("192.168.1.0/24")
		cfg.security.SetCIDR(t, testCIDR)
		require.Equal(t, testCIDR, cfg.security.CIDR())
	})

	t.Run("SetCleanupInterval", func(t *testing.T) {
		expected := 10 * time.Millisecond
		cfg.ipRateLimit.SetCleanupInterval(t, expected)
		require.Equal(t, expected, cfg.ipRateLimit.CleanupInterval())
	})

	t.Run("SetHeaderName", func(t *testing.T) {
		expected := "X-TEST-AUTH"
		cfg.auth.SetHeaderName(t, expected)
		require.Equal(t, expected, cfg.auth.HeaderName())
	})

	t.Run("SetAllowedHeaderBearer", func(t *testing.T) {
		expected := true
		cfg.auth.SetAllowedHeaderBearer(t, expected)
		require.Equal(t, expected, cfg.auth.AllowedHeaderBearer())
	})
}
