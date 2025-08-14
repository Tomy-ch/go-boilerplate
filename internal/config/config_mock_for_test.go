package config

import (
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	// operationSystem
	expectedOSTimeZone = "Asia/Tokyo"
	// server
	expectedEnv     = "test"
	expectedAppMode = DevelopmentMode
	expectedHost    = "localhost"
	expectedPort    = 8080
	// database
	expectedDBHost     = "postgres-db"
	expectedDBPort     = 5432
	expectedDBUser     = "postgres"
	expectedDBPassword = "postgres-password"
	expectedDBName     = "test"
	expectedDBSSLMode  = "disable"
	// dbconnection
	expectedDBMaxOpenConns   = 10
	expectedDBMaxIdleConns   = 5
	expectedDBMaxLifetimeStr = "60s"
	expectedDBMaxLifetime    = time.Duration(60) * time.Second
	expectedDBMaxIdleTimeStr = "30s"
	expectedDBMaxIdleTime    = time.Duration(30) * time.Second
	// security
	expectedAllowedOrigins = "http://localhost,https://example.com"
	expectedCIDRStr        = "192.168.0.0/24"
)

// mockConfig は、テスト用のConfigを返します。
func mockConfig(t *testing.T) Config {
	t.Helper()
	_, expectedCIDR, err := net.ParseCIDR(expectedCIDRStr)
	require.NoError(t, err)

	return Config{
		os: operationSystem{
			timezone: expectedOSTimeZone,
		},
		server: server{
			serverEnv: expectedEnv,
			appMode:   expectedAppMode,
			host:      expectedHost,
			port:      expectedPort,
		},
		database: database{
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
}

// mockLoader は、テスト用のLoaderを返します。
func mockLoader(t *testing.T) Loader {
	t.Helper()

	return Loader{
		OS: OperationSystem{
			Timezone: expectedOSTimeZone,
		},
		Server: Server{
			ServerEnv: expectedEnv,
			AppMode:   expectedAppMode,
			Host:      expectedHost,
			Port:      expectedPort,
		},
		Database: Database{
			Host:     expectedDBHost,
			Port:     expectedDBPort,
			User:     expectedDBUser,
			Password: expectedDBPassword,
			Name:     expectedDBName,
			SSLMode:  expectedDBSSLMode,
		},
		Security: Security{
			AllowedOrigins: strings.Split(expectedAllowedOrigins, ","),
			CIDR:           expectedCIDRStr,
		},
	}
}

// setEnv は、テスト用の環境変数を設定します。
func setEnv(t *testing.T) {
	t.Helper()
	t.Setenv("OS_TZ", expectedOSTimeZone)
	t.Setenv("SERVER_ENV", expectedEnv)
	t.Setenv("SERVER_APP_MODE", expectedAppMode)
	t.Setenv("SERVER_HOST", expectedHost)
	t.Setenv("SERVER_PORT", strconv.Itoa(expectedPort))
	t.Setenv("DB_HOST", expectedDBHost)
	t.Setenv("DB_PORT", strconv.Itoa(expectedDBPort))
	t.Setenv("DB_USER", expectedDBUser)
	t.Setenv("DB_PASSWORD", expectedDBPassword)
	t.Setenv("DB_NAME", expectedDBName)
	t.Setenv("DB_SSL_MODE", expectedDBSSLMode)
	t.Setenv("DB_CONN_MAX_OPEN", strconv.Itoa(expectedDBMaxOpenConns))
	t.Setenv("DB_CONN_MAX_IDLE", strconv.Itoa(expectedDBMaxIdleConns))
	t.Setenv("DB_CONN_MAX_LIFETIME", expectedDBMaxLifetimeStr)
	t.Setenv("DB_CONN_MAX_IDLE_TIME", expectedDBMaxIdleTimeStr)
	t.Setenv("SECURITY_CIDR", expectedCIDRStr)
	t.Setenv("SECURITY_ALLOWED_ORIGINS", expectedAllowedOrigins)
}
