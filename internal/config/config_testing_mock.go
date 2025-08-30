package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	// operationSystem
	expectedOSTimeZone = "Asia/Tokyo"
	// server
	expectedApplicationEnv             = "test"
	expectedApplicationMode            = DevelopmentMode
	expectedHost                       = "localhost"
	expectedPort                       = 8080
	expectedServerShutdownTimeoutCount = 30
	expectedServerShutdownTimeoutStr   = fmt.Sprintf("%ds", expectedServerShutdownTimeoutCount)
	expectedServerShutdownTimeout      = time.Duration(expectedServerShutdownTimeoutCount) * time.Second
	expectedAppShutdownTimeoutCount    = 60
	expectedAppShutdownTimeoutStr      = fmt.Sprintf("%ds", expectedAppShutdownTimeoutCount)
	expectedAppShutdownTimeout         = time.Duration(expectedAppShutdownTimeoutCount) * time.Second
	// database
	expectedDBDriver   = "pgx"
	expectedDBHost     = "localhost"
	expectedDBPort     = 5432
	expectedDBUser     = "postgres"
	expectedDBPassword = "postgres-password"
	expectedDBName     = "test"
	expectedDBSSLMode  = "disable"
	// dbconnection
	expectedDBMaxOpenConns     = 10
	expectedDBMaxIdleConns     = 5
	expectedDBMaxLifetimeCount = 60
	expectedDBMaxLifetimeStr   = fmt.Sprintf("%ds", expectedDBMaxLifetimeCount)
	expectedDBMaxLifetime      = time.Duration(expectedDBMaxLifetimeCount) * time.Second
	expectedDBMaxIdleTimeCount = 30
	expectedDBMaxIdleTimeStr   = fmt.Sprintf("%ds", expectedDBMaxIdleTimeCount)
	expectedDBMaxIdleTime      = time.Duration(expectedDBMaxIdleTimeCount) * time.Second
	// security
	expectedAllowedOrigins = "http://localhost,https://example.com"
	expectedCIDRStr        = "192.168.0.0/24"
	_, expectedCIDR, _     = net.ParseCIDR(expectedCIDRStr)
)

// MockConfigForTest は、テスト用のConfigを返します。
func MockConfigForTest(t testing.TB) *Config {
	t.Helper()
	return &Config{
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
}

// mockLoader は、テスト用のLoaderを返します。
func mockLoader(t testing.TB) Loader {
	t.Helper()

	return Loader{
		OS: OperationSystem{
			Timezone: expectedOSTimeZone,
		},
		App: Application{
			Env:             expectedApplicationEnv,
			Mode:            expectedApplicationMode,
			ShutdownTimeout: expectedAppShutdownTimeout,
		},
		Server: Server{
			Host:            expectedHost,
			Port:            expectedPort,
			ShutdownTimeout: expectedServerShutdownTimeout,
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
func setEnv(t testing.TB) {
	t.Helper()
	t.Setenv("OS_TZ", expectedOSTimeZone)
	t.Setenv("APP_ENV", expectedApplicationEnv)
	t.Setenv("APP_MODE", expectedApplicationMode)
	t.Setenv("APP_SHUTDOWN_TIMEOUT", expectedAppShutdownTimeoutStr)
	t.Setenv("SERVER_HOST", expectedHost)
	t.Setenv("SERVER_PORT", strconv.Itoa(expectedPort))
	t.Setenv("SERVER_SHUTDOWN_TIMEOUT", expectedServerShutdownTimeoutStr)
	t.Setenv("DB_DRIVER", expectedDBDriver)
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
