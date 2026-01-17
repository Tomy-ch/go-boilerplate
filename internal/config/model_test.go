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
		osCfg := NewOperationSystemConfig(cfg)
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

	t.Run("NewObservabilityConfig", func(t *testing.T) {
		t.Parallel()
		observabilityCfg := NewObservabilityConfig(cfg)
		require.Equal(t, &cfg.observability, observabilityCfg)
	})

	t.Run("NewObservabilityConfig", func(t *testing.T) {
		t.Parallel()
		observabilityCfg := NewObservabilityConfig(cfg)
		require.Equal(t, &cfg.observability, observabilityCfg)
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
		require.Equal(t, &cfg.dbconnection, dbConnCfg)
	})

	t.Run("NewSecurityConfig", func(t *testing.T) {
		t.Parallel()
		securityCfg := NewSecurityConfig(cfg)
		require.Equal(t, &cfg.security, securityCfg)
	})

	t.Run("NewSecureCookieConfig", func(t *testing.T) {
		t.Parallel()
		secureCookieCfg := NewSecureCookieConfig(cfg)
		require.Equal(t, &cfg.secureCookie, secureCookieCfg)
	})

	t.Run("NewAuthConfig", func(t *testing.T) {
		t.Parallel()
		authCfg := NewAuthConfig(cfg)
		require.Equal(t, &cfg.auth, authCfg)
	})

	t.Run("NewIPRateLimitConfig", func(t *testing.T) {
		t.Parallel()
		ipRateLimitCfg := NewIPRateLimitConfig(cfg)
		require.Equal(t, &cfg.ipRateLimit, ipRateLimitCfg)
	})
}

func TestGetterMethods(t *testing.T) {
	t.Parallel()

	cfg := MockConfigForTest(t)

	t.Run("OperatingSystem", func(t *testing.T) {
		t.Parallel()
		os := cfg.os
		t.Run("TimeZone", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedOSTimeZone, os.TimeZone())
		})
	})

	t.Run("Server", func(t *testing.T) {
		t.Parallel()
		server := cfg.server
		t.Run("Host", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedServerHost, server.Host())
		})

		t.Run("Port", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedServerPort, server.Port())
		})

		t.Run("ReadHeaderTimeout", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedServerReadHeaderTimeout, server.ReadHeaderTimeout())
		})

		t.Run("ReadTimeout", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedServerReadTimeout, server.ReadTimeout())
		})

		t.Run("WriteTimeout", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedServerWriteTimeout, server.WriteTimeout())
		})

		t.Run("IdleTimeout", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedServerIdleTimeout, server.IdleTimeout())
		})
	})

	t.Run("Metrics", func(t *testing.T) {
		t.Parallel()
		metrics := cfg.metrics

		t.Run("Host", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedMetricsHost, metrics.Host())
		})

		t.Run("Port", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedMetricsPort, metrics.Port())
		})
	})

	t.Run("Observability", func(t *testing.T) {
		t.Parallel()
		observability := cfg.observability

		t.Run("Enabled", func(t *testing.T) {
			t.Parallel()
			require.True(t, observability.Enabled())
		})

		t.Run("TargetStatusCodes", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedObservabilityTargetStatusCodes, observability.TargetStatusCodes())
		})

		t.Run("TargetStatusCodeSet", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedObservabilityTargetStatusCodeSet, observability.TargetStatusCodeSet())
		})
	})

	t.Run("Application", func(t *testing.T) {
		t.Parallel()
		app := cfg.app
		t.Run("Env", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedApplicationEnv, app.Env())
		})

		t.Run("Name", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedApplicationName, app.Name())
		})

		t.Run("Mode", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedApplicationMode, app.Mode())
		})

		t.Run("ShutdownTimeout", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedAppShutdownTimeout, app.ShutdownTimeout())
		})
	})

	t.Run("Database", func(t *testing.T) {
		t.Parallel()
		database := cfg.database
		t.Run("Driver", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBDriver, database.Driver())
		})

		t.Run("Host", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBHost, database.Host())
		})

		t.Run("Port", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBPort, database.Port())
		})

		t.Run("User", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBUser, database.User())
		})

		t.Run("Password", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBPassword, database.Password())
		})

		t.Run("DBName", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBName, database.DBName())
		})

		t.Run("SSLMode", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBSSLMode, database.SSLMode())
		})

		t.Run("SlowQueryWarnThreshold", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBSlowQueryWarnThreshold, database.SlowQueryWarnThreshold())
		})
	})

	t.Run("DBConnection", func(t *testing.T) {
		t.Parallel()
		connection := cfg.dbconnection
		t.Run("MaxOpenConns", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBMaxOpenConns, connection.MaxOpenConns())
		})

		t.Run("MaxIdleConns", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBMaxIdleConns, connection.MaxIdleConns())
		})

		t.Run("MaxLifetime", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBMaxLifetime, connection.MaxLifetime())
		})

		t.Run("MaxIdleTime", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedDBMaxIdleTime, connection.MaxIdleTime())
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

		t.Run("ContentTypeNosniff", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedContentTypeNosniff, security.ContentTypeNosniff())
		})

		t.Run("XFrameOptions", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedXFrameOptions, security.XFrameOptions())
		})

		t.Run("HSTSMaxAge", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedHSTSMaxAge, security.HSTSMaxAge())
		})

		t.Run("HSTSExcludeSubdomains", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedHSTSExcludeSubdomains, security.HSTSExcludeSubdomains())
		})

		t.Run("HSTSPreloadEnabled", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedHSTSPreloadEnabled, security.HSTSPreloadEnabled())
		})

		t.Run("ReferrerPolicy", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedReferrerPolicy, security.ReferrerPolicy())
		})
	})

	t.Run("SecureCookie", func(t *testing.T) {
		t.Parallel()
		secureCookie := cfg.secureCookie

		t.Run("Secure", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedSecureCookieSecure, secureCookie.Secure())
		})

		t.Run("SameSite", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedSecureCookieSameSite, secureCookie.SameSite())
		})

		t.Run("Domain", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedSecureCookieDomain, secureCookie.Domain())
		})
	})

	t.Run("Auth", func(t *testing.T) {
		t.Parallel()
		auth := cfg.auth

		t.Run("CookieName", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedAuthCookieName, auth.CookieName())
		})

		t.Run("HeaderName", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedAuthHeaderName, auth.HeaderName())
		})

		t.Run("AllowedHeaderBearer", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedAuthAllowedHeaderBearer, auth.AllowedHeaderBearer())
		})
	})

	t.Run("IPRateLimit", func(t *testing.T) {
		t.Parallel()
		ipRateLimit := cfg.ipRateLimit

		t.Run("Enabled", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedIPRateLimitEnabled, ipRateLimit.Enabled())
		})

		t.Run("Requests", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedIPRateLimitRequests, ipRateLimit.Requests())
		})

		t.Run("Per", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedIPRateLimitPer, ipRateLimit.Per())
		})

		t.Run("Burst", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedIPRateLimitBurst, ipRateLimit.Burst())
		})

		t.Run("TTL", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedIPRateLimitTTL, ipRateLimit.TTL())
		})

		t.Run("CleanupInterval", func(t *testing.T) {
			t.Parallel()
			require.Equal(t, expectedIPRateLimitCleanupInterval, ipRateLimit.CleanupInterval())
		})
	})
}

func Test_ApplicationConfig_IsProductionMode(t *testing.T) {
	t.Parallel()
	t.Run("本番環境モードの場合", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.app.mode = ProductionMode
		require.True(t, cfg.app.IsProductionMode())
	})

	t.Run("開発環境モードの場合", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.app.mode = DevelopmentMode
		require.False(t, cfg.app.IsProductionMode())
	})
}

func Test_ApplicationConfig_IsDevelopmentMode(t *testing.T) {
	t.Parallel()
	t.Run("開発環境モードの場合", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.app.mode = DevelopmentMode
		require.True(t, cfg.app.IsDevelopmentMode())
	})

	t.Run("本番環境モードの場合", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.app.mode = ProductionMode
		require.False(t, cfg.app.IsDevelopmentMode())
	})
}

func Test_DatabaseConfig_DSN(t *testing.T) {
	t.Parallel()

	cfg := MockConfigForTest(t)

	t.Run("DSN", func(t *testing.T) {
		t.Parallel()
		expectedURL := fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s",
			expectedDBUser,
			expectedDBPassword,
			expectedDBHost,
			expectedDBPort,
			expectedDBName,
			expectedDBSSLMode,
		)
		require.Equal(t, expectedURL, cfg.database.DSN())
	})
}

func Test_DatabaseConfig_DSNWithTimeZone(t *testing.T) {
	t.Parallel()

	cfg := MockConfigForTest(t)

	t.Run("DSNWithTimeZone", func(t *testing.T) {
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
		require.Equal(t, expectedURL, cfg.database.DSNWithTimeZone(&cfg.os))
	})
}

func Test_IPRateLimitConfig_Limit(t *testing.T) {
	t.Parallel()

	cfg := MockConfigForTest(t)
	ipRateLimitCfg := cfg.ipRateLimit

	t.Run("Limit", func(t *testing.T) {
		t.Parallel()
		delta := float64(0.0001)
		expected := float64(expectedIPRateLimitRequests) / expectedIPRateLimitPer.Seconds()
		actual := ipRateLimitCfg.Limit()

		require.InEpsilon(t, expected, actual, delta)
	})
}
