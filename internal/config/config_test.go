package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("configに必要な環境変数が全て設定されている場合", func(t *testing.T) {
			setEnvVarsForTesting(t)
			expected := &Config{
				os: OperationSystemConfig{
					timezone: expectedOSTimeZone,
				},
				app: ApplicationConfig{
					env:             expectedApplicationEnv,
					mode:            expectedApplicationMode,
					shutdownTimeout: expectedAppShutdownTimeout,
				},
				server: ServerConfig{
					host:              expectedServerHost,
					port:              expectedServerPort,
					readHeaderTimeout: expectedServerReadHeaderTimeout,
					readTimeout:       expectedServerReadTimeout,
					writeTimeout:      expectedServerWriteTimeout,
					idleTimeout:       expectedServerIdleTimeout,
				},
				metrics: MetricsConfig{
					host: expectedMetricsHost,
					port: expectedMetricsPort,
				},
				observability: ObservabilityConfig{
					enabled:           expectedObservabilityEnabled,
					targetStatusCodes: expectedObservabilityTargetStatusCodes,
				},
				database: DatabaseConfig{
					driver:                 expectedDBDriver,
					host:                   expectedDBHost,
					port:                   expectedDBPort,
					user:                   expectedDBUser,
					password:               expectedDBPassword,
					name:                   expectedDBName,
					sslMode:                expectedDBSSLMode,
					slowQueryWarnThreshold: expectedDBSlowQueryWarnThreshold,
				},
				dbconnection: DBConnectionConfig{
					maxOpenConns: expectedDBMaxOpenConns,
					maxIdleConns: expectedDBMaxIdleConns,
					maxLifetime:  expectedDBMaxLifetime,
					maxIdleTime:  expectedDBMaxIdleTime,
				},
				security: SecurityConfig{
					allowedOrigins:        strings.Split(expectedAllowedOrigins, ","),
					cidr:                  expectedCIDR,
					contentTypeNosniff:    expectedContentTypeNosniff,
					xFrameOptions:         expectedXFrameOptions,
					hstsMaxAge:            expectedHSTSMaxAge,
					hstsExcludeSubdomains: expectedHSTSExcludeSubdomains,
					hstsPreloadEnabled:    expectedHSTSPreloadEnabled,
					referrerPolicy:        expectedReferrerPolicy,
				},
			}

			actual, err := New()

			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("configに必要な環境変数が不足している場合", func(t *testing.T) {
			t.Setenv("SERVER_ENV", expectedApplicationEnv)

			actual, err := New()
			require.Nil(t, actual)
			require.ErrorContains(t, err, "APP_MODE")
		})

		t.Run("バリデート結果がエラーの場合", func(t *testing.T) {
			setEnvVarsForTesting(t)
			t.Setenv("APP_MODE", "invalid_env")

			actual, err := New()
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidAppMode)
		})
	})
}

func Test_validateConfig(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		cfg := mockLoader(t)

		actual, err := validateConfig(cfg)
		require.NotNil(t, actual)
		require.NoError(t, err)
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("アプリケーション設定でエラーが発生する場合、エラーが返されること", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.App.Mode = "invalid_mode" // 無効なアプリケーションモード

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.Error(t, err)
		})

		t.Run("サーバー設定でエラーが発生する場合、エラーが返されること", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Server.ReadHeaderTimeout = 0 // 無効なReadHeaderTimeout

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.Error(t, err)
		})

		t.Run("DB設定でエラーが発生する場合、エラーが返されること", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Database.SlowQueryWarnThreshold = -1 // 無効なスロークエリ警告閾値

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.Error(t, err)
		})

		t.Run("セキュリティ設定でエラーが発生する場合、エラーが返されること", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Security.CIDR = "invalid_cidr" // 無効なCIDR

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.Error(t, err)
		})
	})
}

func Test_validateApplicationConfig(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		cfg := mockLoader(t)
		err := validateApplicationConfig(cfg.App)
		require.NoError(t, err)
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("無効なアプリケーションモード", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.App.Mode = "invalid_mode" // 無効なアプリケーションモード

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidAppMode)
		})
	})
}

func Test_validateServerConfig(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		cfg := mockLoader(t)
		err := validateServerConfig(cfg.Server)
		require.NoError(t, err)
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("無効なポート番号", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Server.Port = MaxPort + 1 // 無効なポート番号

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidPortRange)
		})

		t.Run("ReadHeaderTimeoutが無効な場合", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Server.ReadHeaderTimeout = 0 // 無効なReadHeaderTimeout

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidReadHeaderTimeout)
		})

		t.Run("ReadTimeoutが無効な場合", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Server.ReadTimeout = 0 // 無効なReadTimeout

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidReadTimeout)
		})

		t.Run("WriteTimeoutが無効な場合", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Server.WriteTimeout = 0 // 無効なWriteTimeout

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidWriteTimeout)
		})

		t.Run("IdleTimeoutが無効な場合", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Server.IdleTimeout = 0 // 無効なIdleTimeout

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidIdleTimeout)
		})

		t.Run("ReadHeaderTimeoutがReadTimeoutを超えている場合", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Server.ReadHeaderTimeout = cfg.Server.ReadTimeout + cfg.Server.ReadTimeout

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrReadHeaderTimeoutExceedsReadTimeout)
		})
	})
}

func Test_validateDatabaseConfig(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		cfg := mockLoader(t)
		err := validateDatabaseConfig(cfg.Database)
		require.NoError(t, err)
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("無効なスロークエリ警告閾値", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Database.SlowQueryWarnThreshold = -1 // 無効なスロークエリ警告閾値

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidSlowQueryWarnThreshold)
		})
	})
}

func Test_validateSecurityConfig(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		cfg := mockLoader(t)
		cidr, err := validateSecurityConfig(cfg.Security)
		require.NoError(t, err)
		require.NotNil(t, cidr)
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("AllowedOriginsが空の場合", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Security.AllowedOrigins = []string{} // 空のAllowedOrigins

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrEmptyAllowedOrigins)
		})

		t.Run("localhost以外でHTTPが許可されている場合", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Security.AllowedOrigins = []string{"http://example.com"} // localhost以外のHTTP

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrHTTPOnlyAllowedForLocalhost)
		})

		t.Run("CIDRのパースに失敗した場合", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Security.CIDR = "invalid_cidr" // 無効なCIDR

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrFailedToParseCIDR)
		})
	})
}
