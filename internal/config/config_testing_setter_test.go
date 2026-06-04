package config

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigTestingSetters(t *testing.T) {
	cfg := MockConfigForTest(t)

	t.Run("SetAppMode", func(t *testing.T) {
		expected := "test-mode"
		cfg.app.SetApplicationMode(t, expected)
		assert.Equal(t, expected, cfg.app.Mode())
	})

	t.Run("SetAppEnv", func(t *testing.T) {
		expected := "test-env"
		cfg.app.SetApplicationEnv(t, expected)
		assert.Equal(t, expected, cfg.app.Env())
	})

	t.Run("SetServerPort", func(t *testing.T) {
		expected := 8081
		cfg.server.SetServerPort(t, expected)
		assert.Equal(t, expected, cfg.server.Port())
	})

	t.Run("SetObservabilityMaskedDBQueryArgs", func(t *testing.T) {
		expected := true
		cfg.observability.SetObservabilityMaskedDBQueryArgs(t, expected)
		assert.Equal(t, expected, cfg.observability.MaskedDBQueryArgs())
	})

	t.Run("SetDatabaseHost", func(t *testing.T) {
		expected := "test-host"
		cfg.database.SetDatabaseHost(t, expected)
		assert.Equal(t, expected, cfg.database.Host())
	})

	t.Run("SetDatabaseName", func(t *testing.T) {
		expected := "test-name"
		cfg.database.SetDatabaseName(t, expected)
		assert.Equal(t, expected, cfg.database.DBName())
	})

	t.Run("SetMaxConns", func(t *testing.T) {
		expected := int32(20)
		cfg.dbconnection.SetMaxConns(t, expected)
		assert.Equal(t, expected, cfg.dbconnection.MaxConns())
	})

	t.Run("SetCIDR", func(t *testing.T) {
		_, testCIDR, _ := net.ParseCIDR("192.168.1.0/24")
		cfg.security.SetCIDR(t, testCIDR)
		assert.Equal(t, testCIDR, cfg.security.CIDR())
	})

	t.Run("SetCleanupInterval", func(t *testing.T) {
		expected := 10 * time.Millisecond
		cfg.ipRateLimit.SetCleanupInterval(t, expected)
		assert.Equal(t, expected, cfg.ipRateLimit.CleanupInterval())
	})

	t.Run("SetHeaderName", func(t *testing.T) {
		expected := "X-TEST-AUTH"
		cfg.auth.SetHeaderName(t, expected)
		assert.Equal(t, expected, cfg.auth.HeaderName())
	})

	t.Run("SetAllowedHeaderBearer", func(t *testing.T) {
		expected := true
		cfg.auth.SetAllowedHeaderBearer(t, expected)
		assert.Equal(t, expected, cfg.auth.AllowedHeaderBearer())
	})
}
