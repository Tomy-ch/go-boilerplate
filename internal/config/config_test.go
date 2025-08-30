package config

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("configに必要な環境変数が全て設定されている場合", func(t *testing.T) {
			setEnv(t)
			expected := &Config{
				os: operationSystem{
					timezone: expectedOSTimeZone,
				},
				app: application{
					env:             expectedApplicationEnv,
					mode:            expectedApplicationMode,
					shutdownTimeout: expectedAppShutdownTimeout,
				},
				server: server{
					host:            expectedHost,
					port:            expectedPort,
					shutdownTimeout: expectedServerShutdownTimeout,
				},
				database: database{
					driver:   expectedDBDriver,
					host:     expectedDBHost,
					port:     expectedDBPort,
					user:     expectedDBUser,
					password: expectedDBPassword,
					name:     expectedDBName,
					sslMode:  expectedDBSSLMode,
					connection: connection{
						maxOpenConns: expectedDBMaxOpenConns,
						maxIdleConns: expectedDBMaxIdleConns,
						maxLifetime:  expectedDBMaxLifetime,
						maxIdleTime:  expectedDBMaxIdleTime,
					},
				},
				security: security{
					allowedOrigins: strings.Split(expectedAllowedOrigins, ","),
					cidr:           expectedCIDR,
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
			setEnv(t)
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
		t.Run("無効なポート番号", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Server.Port = MaxPort + 1 // 無効なポート番号

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidPortRange)
		})

		t.Run("AllowedOriginsが空の場合", func(t *testing.T) {
			cfg := mockLoader(t)
			cfg.Security.AllowedOrigins = []string{} // 空のAllowedOrigins

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrEmptyAllowedOrigins)
		})

		t.Run("localhost以外でHTTPが許可されている場合", func(t *testing.T) {
			cfg := mockLoader(t)
			cfg.Security.AllowedOrigins = []string{"http://example.com"} // localhost以外のHTTP

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrHTTPOnlyAllowedForLocalhost)
		})

		t.Run("無効なアプリケーションモード", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.App.Mode = "invalid_mode" // 無効なアプリケーションモード

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidAppMode)
		})

		t.Run("サーバーのシャットダウンタイムアウトがアプリケーションのシャットダウンタイムアウトを超えている場合", func(t *testing.T) {
			cfg := mockLoader(t)
			cfg.Server.ShutdownTimeout = cfg.App.ShutdownTimeout + time.Duration(1*time.Second)

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrServerErrShutdownTimeoutExceedsApplication)
		})

		t.Run("CIDRのパースに失敗した場合", func(t *testing.T) {
			cfg := mockLoader(t)
			cfg.Security.CIDR = "invalid_cidr" // 無効なCIDR

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrFailedToParseCIDR)
		})
	})
}

func TestIsAppProductionMode(t *testing.T) {
	t.Parallel()
	t.Run("本番環境モードの場合", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.app.mode = ProductionMode
		require.True(t, cfg.IsAppProductionMode())
	})

	t.Run("開発環境モードの場合", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.app.mode = DevelopmentMode
		require.False(t, cfg.IsAppProductionMode())
	})
}

func TestIsAppDevelopmentMode(t *testing.T) {
	t.Parallel()
	t.Run("開発環境モードの場合", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.app.mode = DevelopmentMode
		require.True(t, cfg.IsAppDevelopmentMode())
	})

	t.Run("本番環境モードの場合", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.app.mode = ProductionMode
		require.False(t, cfg.IsAppDevelopmentMode())
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
		require.Equal(t, expectedURL, cfg.DatabaseDSN())
	})
}
