package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstructor(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		cfg := MockConfigForTest(t)

		t.Run("NewOSConfig", func(t *testing.T) {
			t.Parallel()
			osCfg := NewOperatingSystemConfig(cfg)
			assert.Same(t, &cfg.os, osCfg)
		})

		t.Run("NewServerConfig", func(t *testing.T) {
			t.Parallel()
			serverCfg := NewServerConfig(cfg)
			assert.Same(t, &cfg.server, serverCfg)
		})

		t.Run("NewMetricsConfig", func(t *testing.T) {
			t.Parallel()
			metricsCfg := NewMetricsConfig(cfg)
			assert.Same(t, &cfg.metrics, metricsCfg)
		})

		t.Run("NewObservabilityConfig", func(t *testing.T) {
			t.Parallel()
			observabilityCfg := NewObservabilityConfig(cfg)
			assert.Same(t, &cfg.observability, observabilityCfg)
		})

		t.Run("NewObservabilityConfig", func(t *testing.T) {
			t.Parallel()
			observabilityCfg := NewObservabilityConfig(cfg)
			assert.Same(t, &cfg.observability, observabilityCfg)
		})

		t.Run("NewApplicationConfig", func(t *testing.T) {
			t.Parallel()
			appCfg := NewApplicationConfig(cfg)
			assert.Same(t, &cfg.app, appCfg)
		})

		t.Run("NewDatabaseConfig", func(t *testing.T) {
			t.Parallel()
			dbCfg := NewDatabaseConfig(cfg)
			assert.Same(t, &cfg.database, dbCfg)
		})

		t.Run("NewDBConnectionConfig", func(t *testing.T) {
			t.Parallel()
			dbConnCfg := NewDBConnectionConfig(cfg)
			assert.Same(t, &cfg.dbconnection, dbConnCfg)
		})

		t.Run("NewSecurityConfig", func(t *testing.T) {
			t.Parallel()
			securityCfg := NewSecurityConfig(cfg)
			assert.Same(t, &cfg.security, securityCfg)
		})

		t.Run("NewSecureCookieConfig", func(t *testing.T) {
			t.Parallel()
			secureCookieCfg := NewSecureCookieConfig(cfg)
			assert.Same(t, &cfg.secureCookie, secureCookieCfg)
		})

		t.Run("NewAuthConfig", func(t *testing.T) {
			t.Parallel()
			authCfg := NewAuthConfig(cfg)
			assert.Same(t, &cfg.auth, authCfg)
		})
	})
}

func TestGetterMethods(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		cfg := MockConfigForTest(t)

		t.Run("OperatingSystem", func(t *testing.T) {
			t.Parallel()
			os := cfg.os
			t.Run("TimeZone", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedOSTimeZone, os.TimeZone())
			})
		})

		t.Run("Server", func(t *testing.T) {
			t.Parallel()
			server := cfg.server
			t.Run("Host", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedServerHost, server.Host())
			})

			t.Run("Port", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedServerPort, server.Port())
			})

			t.Run("ReadHeaderTimeout", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedServerReadHeaderTimeout, server.ReadHeaderTimeout())
			})

			t.Run("ReadTimeout", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedServerReadTimeout, server.ReadTimeout())
			})

			t.Run("WriteTimeout", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedServerWriteTimeout, server.WriteTimeout())
			})

			t.Run("IdleTimeout", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedServerIdleTimeout, server.IdleTimeout())
			})
		})

		t.Run("Metrics", func(t *testing.T) {
			t.Parallel()
			metrics := cfg.metrics

			t.Run("Host", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedMetricsHost, metrics.Host())
			})

			t.Run("Port", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedMetricsPort, metrics.Port())
			})

			t.Run("UserName", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedMetricsUserName, metrics.UserName())
			})

			t.Run("Password", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedMetricsPassword, metrics.Password())
			})
		})

		t.Run("Observability", func(t *testing.T) {
			t.Parallel()
			observability := cfg.observability

			t.Run("Enabled", func(t *testing.T) {
				t.Parallel()
				assert.True(t, observability.Enabled())
			})

			t.Run("MaskedDBQueryArgs", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedObservabilityMaskedDBQueryArgs, observability.MaskedDBQueryArgs())
			})

			t.Run("TargetStatusCodeSet", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedObservabilityTargetStatusCodeSet, observability.TargetStatusCodeSet())
			})
		})

		t.Run("Application", func(t *testing.T) {
			t.Parallel()
			app := cfg.app
			t.Run("Env", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedApplicationEnv, app.Env())
			})

			t.Run("Name", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedApplicationName, app.Name())
			})

			t.Run("Mode", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedApplicationMode, app.Mode())
			})

			t.Run("ShutdownTimeout", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedAppShutdownTimeout, app.ShutdownTimeout())
			})
		})

		t.Run("Database", func(t *testing.T) {
			t.Parallel()
			database := cfg.database
			t.Run("Driver", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBDriver, database.Driver())
			})

			t.Run("Host", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBHost, database.Host())
			})

			t.Run("Port", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBPort, database.Port())
			})

			t.Run("User", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBUser, database.User())
			})

			t.Run("Password", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBPassword, database.Password())
			})

			t.Run("DBName", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBName, database.DBName())
			})

			t.Run("SSLMode", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBSSLMode, database.SSLMode())
			})

			t.Run("PingTimeout", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBPingTimeout, database.PingTimeout())
			})

			t.Run("SlowQueryWarnThreshold", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBSlowQueryWarnThreshold, database.SlowQueryWarnThreshold())
			})
		})

		t.Run("DBConnection", func(t *testing.T) {
			t.Parallel()
			connection := cfg.dbconnection
			t.Run("MaxConns", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBMaxConnsInt32, connection.MaxConns())
			})

			t.Run("MinConns", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBMinConnsInt32, connection.MinConns())
			})

			t.Run("MaxLifetime", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBMaxLifetime, connection.MaxLifetime())
			})

			t.Run("MaxIdleTime", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBMaxIdleTime, connection.MaxIdleTime())
			})
		})

		t.Run("Security", func(t *testing.T) {
			t.Parallel()
			security := cfg.security
			t.Run("AllowedOrigins", func(t *testing.T) {
				t.Parallel()
				assert.Equal(
					t,
					strings.Split(expectedAllowedOrigins, ","),
					security.AllowedOrigins(),
				)
			})

			t.Run("CIDR", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedCIDR, security.CIDR())
			})

			t.Run("ContentTypeNosniff", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedContentTypeNosniff, security.ContentTypeNosniff())
			})

			t.Run("XFrameOptions", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedXFrameOptions, security.XFrameOptions())
			})

			t.Run("HSTSMaxAge", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedHSTSMaxAge, security.HSTSMaxAge())
			})

			t.Run("HSTSExcludeSubdomains", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedHSTSExcludeSubdomains, security.HSTSExcludeSubdomains())
			})

			t.Run("HSTSPreloadEnabled", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedHSTSPreloadEnabled, security.HSTSPreloadEnabled())
			})

			t.Run("ReferrerPolicy", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedReferrerPolicy, security.ReferrerPolicy())
			})

			t.Run("BcryptCost", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedBcryptCost, security.BcryptCost())
			})
		})

		t.Run("SecureCookie", func(t *testing.T) {
			t.Parallel()
			secureCookie := cfg.secureCookie

			t.Run("Secure", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedSecureCookieSecure, secureCookie.Secure())
			})

			t.Run("SameSite", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedSecureCookieSameSite, secureCookie.SameSite())
			})

			t.Run("Domain", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedSecureCookieDomain, secureCookie.Domain())
			})
		})

		t.Run("Auth", func(t *testing.T) {
			t.Parallel()
			auth := cfg.auth

			t.Run("CookieName", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedAuthCookieName, auth.CookieName())
			})

			t.Run("HeaderName", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedAuthHeaderName, auth.HeaderName())
			})

			t.Run("AllowedHeaderBearer", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedAuthAllowedHeaderBearer, auth.AllowedHeaderBearer())
			})
		})
	})
}

func Test_ApplicationConfig_IsProductionMode(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("本番環境モードの場合", func(t *testing.T) {
			t.Parallel()
			cfg := Config{}
			cfg.app.mode = ProductionMode
			assert.True(t, cfg.app.IsProductionMode())
		})

		t.Run("開発環境モードの場合", func(t *testing.T) {
			t.Parallel()
			cfg := Config{}
			cfg.app.mode = DevelopmentMode
			assert.False(t, cfg.app.IsProductionMode())
		})
	})
}

func Test_ApplicationConfig_IsDevelopmentMode(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("開発環境モードの場合", func(t *testing.T) {
			t.Parallel()
			cfg := Config{}
			cfg.app.mode = DevelopmentMode
			assert.True(t, cfg.app.IsDevelopmentMode())
		})

		t.Run("本番環境モードの場合", func(t *testing.T) {
			t.Parallel()
			cfg := Config{}
			cfg.app.mode = ProductionMode
			assert.False(t, cfg.app.IsDevelopmentMode())
		})
	})
}

func TestSecurityConfig_CIDR(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("cidrがnilの場合_nilを返す", func(t *testing.T) {
			t.Parallel()
			s := &SecurityConfig{}
			assert.Nil(t, s.CIDR())
		})
	})
}

func TestSecureCookieConfig_Secure(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("secureがnilの場合_nilを返す", func(t *testing.T) {
			t.Parallel()
			s := &SecureCookieConfig{}
			assert.Nil(t, s.Secure())
		})
	})
}
