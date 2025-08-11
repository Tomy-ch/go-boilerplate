package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	// environment
	expectedEnv := "test"
	expectedAppMode := "development"
	// server
	expectedHost := "localhost"
	expectedPort := 8080
	// database
	expectedDBHost := "postgres-db"
	expectedDBPort := 5432
	expectedDBUser := "postgres"
	expectedDBPassword := "postgres-password"
	expectedDBName := "test"
	expectedDBSSLMode := "disable"
	// security
	expectedAllowedOrigins := "http://localhost,https://example.com"
	expectedCIDRStr := "192.168.0.0/24"
	_, expectedCIDR, err := net.ParseCIDR(expectedCIDRStr)
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Run("configに必要な環境変数が全て設定されている場合", func(t *testing.T) {
			t.Setenv("ENV", expectedEnv)
			t.Setenv("APP_MODE", expectedAppMode)
			t.Setenv("HOST", expectedHost)
			t.Setenv("PORT", strconv.Itoa(expectedPort))
			t.Setenv("DB_HOST", expectedDBHost)
			t.Setenv("DB_PORT", strconv.Itoa(expectedDBPort))
			t.Setenv("DB_USER", expectedDBUser)
			t.Setenv("DB_PASSWORD", expectedDBPassword)
			t.Setenv("DB_NAME", expectedDBName)
			t.Setenv("DB_SSLMODE", expectedDBSSLMode)
			t.Setenv("SECURITY_CIDR", expectedCIDRStr)
			t.Setenv("SECURITY_ALLOWED_ORIGINS", expectedAllowedOrigins)

			expected := &Config{
				environment: environment{
					serverEnv: expectedEnv,
					appMode:   expectedAppMode,
				},
				server: server{
					host: expectedHost,
					port: expectedPort,
				},
				database: database{
					host:     expectedDBHost,
					port:     expectedDBPort,
					user:     expectedDBUser,
					password: expectedDBPassword,
					name:     expectedDBName,
					sslMode:  expectedDBSSLMode,
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
			t.Setenv("ENV", expectedEnv)

			actual, err := New()
			require.Nil(t, actual)
			require.ErrorContains(t, err, "APP_MODE")
		})

		t.Run("バリデート結果がエラーの場合", func(t *testing.T) {
			t.Setenv("ENV", expectedEnv)
			t.Setenv("APP_MODE", "invalid_env")
			t.Setenv("HOST", expectedHost)
			t.Setenv("PORT", strconv.Itoa(expectedPort))
			t.Setenv("DB_HOST", expectedDBHost)
			t.Setenv("DB_PORT", strconv.Itoa(expectedDBPort))
			t.Setenv("DB_USER", expectedDBUser)
			t.Setenv("DB_PASSWORD", expectedDBPassword)
			t.Setenv("DB_NAME", expectedDBName)
			t.Setenv("DB_SSLMODE", expectedDBSSLMode)
			t.Setenv("SECURITY_CIDR", expectedCIDRStr)
			t.Setenv("SECURITY_ALLOWED_ORIGINS", expectedAllowedOrigins)

			actual, err := New()
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidAppMode)
		})
	})
}

func TestGetterMethods(t *testing.T) {
	t.Parallel()

	// environment
	expectedEnv := "test"
	expectedAppMode := "development"
	// server
	expectedHost := "localhost"
	expectedPort := 8080
	// database
	expectedDBHost := "postgres-db"
	expectedDBPort := 5432
	expectedDBUser := "postgres"
	expectedDBPassword := "postgres-password"
	expectedDBName := "test"
	expectedDBSSLMode := "disable"
	// security
	expectedAllowedOrigins := "http://localhost,https://example.com"
	expectedCIDRStr := "192.168.0.0/24"
	_, expectedCIDR, err := net.ParseCIDR(expectedCIDRStr)
	require.NoError(t, err)

	cfg := &Config{
		environment: environment{
			serverEnv: expectedEnv,
			appMode:   expectedAppMode,
		},
		server: server{
			host: expectedHost,
			port: expectedPort,
		},
		database: database{
			host:     expectedDBHost,
			port:     expectedDBPort,
			user:     expectedDBUser,
			password: expectedDBPassword,
			name:     expectedDBName,
			sslMode:  expectedDBSSLMode,
		},
		security: security{
			allowedOrigins: strings.Split(expectedAllowedOrigins, ","),
			cidr:           expectedCIDR,
		},
	}

	t.Run("ServerHost", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedHost, cfg.ServerHost())
	})

	t.Run("ServerPort", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedPort, cfg.ServerPort())
	})

	t.Run("ServerEnv", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedEnv, cfg.ServerEnv())
	})

	t.Run("AppMode", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, expectedAppMode, cfg.AppMode())
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

	t.Run("DatabaseURL", func(t *testing.T) {
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
		require.Equal(t, expectedURL, cfg.DatabaseURL())
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

func Test_validateConfig(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		cfg := verifiedConfigLoader()

		actual, err := validateConfig(cfg)
		require.NotNil(t, actual)
		require.NoError(t, err)
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()
		t.Run("無効なポート番号", func(t *testing.T) {
			t.Parallel()
			cfg := verifiedConfigLoader()
			cfg.Server.Port = MaxPort + 1 // 無効なポート番号

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidPortRange)
		})
		t.Run("無効なアプリケーションモード", func(t *testing.T) {
			t.Parallel()
			cfg := verifiedConfigLoader()
			cfg.Environment.AppMode = "invalid_mode" // 無効なアプリケーションモード

			actual, err := validateConfig(cfg)
			require.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidAppMode)
		})

		t.Run("CIDRのパースに失敗した場合", func(t *testing.T) {
			cfg := verifiedConfigLoader()
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
		cfg.environment.appMode = ProductionMode
		require.True(t, cfg.IsAppProductionMode())
	})

	t.Run("開発環境モードの場合", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.environment.appMode = DevelopmentMode
		require.False(t, cfg.IsAppProductionMode())
	})
}

func TestIsAppDevelopmentMode(t *testing.T) {
	t.Parallel()
	t.Run("開発環境モードの場合", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.environment.appMode = DevelopmentMode
		require.True(t, cfg.IsAppDevelopmentMode())
	})

	t.Run("本番環境モードの場合", func(t *testing.T) {
		t.Parallel()
		cfg := Config{}
		cfg.environment.appMode = ProductionMode
		require.False(t, cfg.IsAppDevelopmentMode())
	})
}

func verifiedConfigLoader() Loader {
	return Loader{
		Server: Server{
			Host: "localhost",
			Port: 8080,
		},
		Environment: Environment{
			ServerEnv: "test",
			AppMode:   DevelopmentMode,
		},
		Security: Security{
			AllowedOrigins: []string{"http://localhost", "https://example.com"},
			CIDR:           "127.0.0.0/8",
		},
	}
}
