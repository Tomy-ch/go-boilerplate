package config

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConstructor(t *testing.T) {
	t.Parallel()
	cfg := MockConfigForTest(t)

	t.Run("NewOSConfig", func(t *testing.T) {
		t.Parallel()
		osCfg := NewOSConfig(cfg)
		require.Equal(t, &cfg.os, osCfg)
	})

	t.Run("NewServerConfig", func(t *testing.T) {
		t.Parallel()
		serverCfg := NewServerConfig(cfg)
		require.Equal(t, &cfg.server, serverCfg)
	})

	t.Run("NewMetricsConfig", func(t *testing.T) {
		t.Parallel()
		metricsCfg := NewMetricsConfig(cfg)
		require.Equal(t, &cfg.metrics, metricsCfg)
	})

	t.Run("NewApplicationConfig", func(t *testing.T) {
		t.Parallel()
		appCfg := NewApplicationConfig(cfg)
		require.Equal(t, &cfg.app, appCfg)
	})

	t.Run("NewDatabaseConfig", func(t *testing.T) {
		t.Parallel()
		dbCfg := NewDatabaseConfig(cfg)
		require.Equal(t, &cfg.database, dbCfg)
	})

	t.Run("NewDBConnectionConfig", func(t *testing.T) {
		t.Parallel()
		dbConnCfg := NewDBConnectionConfig(cfg)
		require.Equal(t, &cfg.database.connection, dbConnCfg)
	})

	t.Run("NewSecurityConfig", func(t *testing.T) {
		t.Parallel()
		securityCfg := NewSecurityConfig(cfg)
		require.Equal(t, &cfg.security, securityCfg)
	})
}

func TestGetterMethods(t *testing.T) {
	t.Parallel()

	cfg := MockConfigForTest(t)

	t.Run("OperatingSystem", func(t *testing.T) {
		t.Parallel()
		os := cfg.os
		t.Run("OSTimeZone", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedOSTimeZone, os.OSTimeZone())
		})
	})

	t.Run("Server", func(t *testing.T) {
		t.Parallel()
		server := cfg.server
		t.Run("ServerHost", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedServerHost, server.ServerHost())
		})

		t.Run("ServerPort", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedServerPort, server.ServerPort())
		})

		t.Run("ServerShutdownTimeout", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedServerShutdownTimeout, server.ServerShutdownTimeout())
		})
	})

	t.Run("Metrics", func(t *testing.T) {
		t.Parallel()
		metrics := cfg.metrics

		t.Run("MetricsHost", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedMetricsHost, metrics.MetricsHost())
		})

		t.Run("MetricsPort", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedMetricsPort, metrics.MetricsPort())
		})
	})

	t.Run("Application", func(t *testing.T) {
		t.Parallel()
		app := cfg.app
		t.Run("AppEnv", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedApplicationEnv, app.AppEnv())
		})

		t.Run("AppName", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedApplicationName, app.AppName())
		})

		t.Run("AppMode", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedApplicationMode, app.AppMode())
		})

		t.Run("AppShutdownTimeout", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedAppShutdownTimeout, app.AppShutdownTimeout())
		})
	})

	t.Run("Database", func(t *testing.T) {
		t.Parallel()
		database := cfg.database
		t.Run("DatabaseDriver", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBDriver, database.DatabaseDriver())
		})

		t.Run("DatabaseHost", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBHost, database.DatabaseHost())
		})

		t.Run("DatabasePort", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBPort, database.DatabasePort())
		})

		t.Run("DatabaseUser", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBUser, database.DatabaseUser())
		})

		t.Run("DatabasePassword", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBPassword, database.DatabasePassword())
		})

		t.Run("DatabaseName", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBName, database.DatabaseName())
		})

		t.Run("DatabaseSSLMode", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBSSLMode, database.DatabaseSSLMode())
		})
	})

	t.Run("DBConnection", func(t *testing.T) {
		t.Parallel()
		connection := cfg.database.connection
		t.Run("DBMaxOpenConns", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBMaxOpenConns, connection.DBMaxOpenConns())
		})

		t.Run("DBMaxIdleConns", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBMaxIdleConns, connection.DBMaxIdleConns())
		})

		t.Run("DBConnMaxLifetime", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBMaxLifetime, connection.DBConnMaxLifetime())
		})

		t.Run("DBConnMaxIdleTime", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBMaxIdleTime, connection.DBConnMaxIdleTime())
		})
	})

	t.Run("Security", func(t *testing.T) {
		t.Parallel()
		security := cfg.security
		t.Run("AllowedOrigins", func(t *testing.T) {
			t.Parallel()
			require.Equal(
				t,
				strings.Split(expectedAllowedOrigins, ","),
				security.AllowedOrigins(),
			)
		})

		t.Run("CIDR", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedCIDR, security.CIDR())
		})
	})
}

func TestIsAppProductionMode(t *testing.T) {
	t.Parallel()
	t.Run("本番環境モードの場合", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.app.mode = ProductionMode
		require.True(t, cfg.app.IsAppProductionMode())
	})

	t.Run("開発環境モードの場合", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.app.mode = DevelopmentMode
		require.False(t, cfg.app.IsAppProductionMode())
	})
}

func TestIsAppDevelopmentMode(t *testing.T) {
	t.Parallel()
	t.Run("開発環境モードの場合", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.app.mode = DevelopmentMode
		require.True(t, cfg.app.IsAppDevelopmentMode())
	})

	t.Run("本番環境モードの場合", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.app.mode = ProductionMode
		require.False(t, cfg.app.IsAppDevelopmentMode())
	})
}

func TestDatabaseURL(t *testing.T) {
	t.Parallel()

	cfg := MockConfigForTest(t)

	t.Run("DatabaseURL", func(t *testing.T) {
		t.Parallel()
		expectedURL := fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s&timezone=%s",
			expectedDBUser,
			expectedDBPassword,
			expectedDBHost,
			expectedDBPort,
			expectedDBName,
			expectedDBSSLMode,
			url.QueryEscape(expectedOSTimeZone),
		)
		require.Equal(t, expectedURL, cfg.database.DatabaseDSN(&cfg.os))
	})
}
