package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetterMethods(t *testing.T) {
	t.Parallel()

	cfg := MockConfigForTest(t)

	t.Run("OSTimeZone", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedOSTimeZone, cfg.OSTimeZone())
	})

	t.Run("ServerHost", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedHost, cfg.ServerHost())
	})

	t.Run("ServerPort", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedPort, cfg.ServerPort())
	})

	t.Run("ServerShutdownTimeout", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedServerShutdownTimeout, cfg.ServerShutdownTimeout())
	})

	t.Run("AppEnv", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedApplicationEnv, cfg.AppEnv())
	})

	t.Run("AppMode", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedApplicationMode, cfg.AppMode())
	})

	t.Run("AppShutdownTimeout", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedAppShutdownTimeout, cfg.AppShutdownTimeout())
	})

	t.Run("DatabaseDriver", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedDBDriver, cfg.DatabaseDriver())
	})

	t.Run("DatabaseHost", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedDBHost, cfg.DatabaseHost())
	})

	t.Run("DatabasePort", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedDBPort, cfg.DatabasePort())
	})

	t.Run("DatabaseUser", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedDBUser, cfg.DatabaseUser())
	})

	t.Run("DatabasePassword", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedDBPassword, cfg.DatabasePassword())
	})

	t.Run("DatabaseName", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedDBName, cfg.DatabaseName())
	})

	t.Run("DatabaseSSLMode", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedDBSSLMode, cfg.DatabaseSSLMode())
	})

	t.Run("DBMaxOpenConns", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedDBMaxOpenConns, cfg.DBMaxOpenConns())
	})

	t.Run("DBMaxIdleConns", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedDBMaxIdleConns, cfg.DBMaxIdleConns())
	})

	t.Run("DBConnMaxLifetime", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedDBMaxLifetime, cfg.DBConnMaxLifetime())
	})

	t.Run("DBConnMaxIdleTime", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedDBMaxIdleTime, cfg.DBConnMaxIdleTime())
	})

	t.Run("AllowedOrigins", func(t *testing.T) {
		t.Parallel()
		require.Equal(
			t,
			strings.Split(expectedAllowedOrigins, ","),
			cfg.AllowedOrigins(),
		)
	})

	t.Run("CIDR", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedCIDR, cfg.CIDR())
	})
}
