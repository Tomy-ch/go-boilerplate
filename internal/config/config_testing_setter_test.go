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
		cfg.app.SetApplicationMode(t, expected)
		require.Equal(t, expected, cfg.app.Mode())
	})

	t.Run("SetAppEnv", func(t *testing.T) {
		expected := "test-env"
		cfg.app.SetApplicationEnv(t, expected)
		require.Equal(t, expected, cfg.app.Env())
	})

	t.Run("SetServerPort", func(t *testing.T) {
		expected := 8081
		cfg.server.SetServerPort(t, expected)
		require.Equal(t, expected, cfg.server.Port())
	})

	t.Run("SetObservabilityMaskedDBQueryArgs", func(t *testing.T) {
		expected := true
		cfg.observability.SetObservabilityMaskedDBQueryArgs(t, expected)
		require.Equal(t, expected, cfg.observability.MaskedDBQueryArgs())
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

	t.Run("SetDatabasePassword", func(t *testing.T) {
		expected := "test-password"
		cfg.database.SetDatabasePassword(t, expected)
		require.Equal(t, expected, cfg.database.Password())
	})

	t.Run("SetMaxOpenConns", func(t *testing.T) {
		expected := int32(20)
		cfg.dbconnection.SetMaxOpenConns(t, expected)
		require.Equal(t, expected, cfg.dbconnection.MaxOpenConns())
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
