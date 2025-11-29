package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

// 下記の変数は、テスト用の期待値以外に、テスト環境の環境変数設定にも使用されます。
// 変更の際は、テストを必ず実行し、環境変数の設定が正しいことを確認してください。
var (
	// operationSystem
	expectedOSTimeZone = "Asia/Tokyo"
	// application
	expectedApplicationEnv          = "test"
	expectedApplicationName         = "TestApp"
	expectedApplicationMode         = DevelopmentMode
	expectedAppShutdownTimeoutCount = 60
	expectedAppShutdownTimeoutStr   = fmt.Sprintf("%ds", expectedAppShutdownTimeoutCount)
	expectedAppShutdownTimeout      = time.Duration(expectedAppShutdownTimeoutCount) * time.Second
	// server
	expectedServerHost                 = "localhost"
	expectedServerPort                 = 8080
	expectedServerShutdownTimeoutCount = 30
	expectedServerShutdownTimeoutStr   = fmt.Sprintf("%ds", expectedServerShutdownTimeoutCount)
	expectedServerShutdownTimeout      = time.Duration(expectedServerShutdownTimeoutCount) * time.Second
	// metrics
	expectedMetricsHost = "localhost"
	expectedMetricsPort = 6060
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
		os: OperationSystemConfig{
			timezone: expectedOSTimeZone,
		},
		app: ApplicationConfig{
			env:             expectedApplicationEnv,
			name:            expectedApplicationName,
			mode:            expectedApplicationMode,
			shutdownTimeout: expectedAppShutdownTimeout,
		},
		server: ServerConfig{
			host:            expectedServerHost,
			port:            expectedServerPort,
			shutdownTimeout: expectedServerShutdownTimeout,
		},
		metrics: MetricsConfig{
			host: expectedMetricsHost,
			port: expectedMetricsPort,
		},
		database: DatabaseConfig{
			driver:   expectedDBDriver,
			host:     expectedDBHost,
			port:     expectedDBPort,
			user:     expectedDBUser,
			password: expectedDBPassword,
			name:     expectedDBName,
			sslMode:  expectedDBSSLMode,
			connection: DBConnectionConfig{
				maxOpenConns: expectedDBMaxOpenConns,
				maxIdleConns: expectedDBMaxIdleConns,
				maxLifetime:  expectedDBMaxLifetime,
				maxIdleTime:  expectedDBMaxIdleTime,
			},
		},
		security: SecurityConfig{
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
			Name:            expectedApplicationName,
			Mode:            expectedApplicationMode,
			ShutdownTimeout: expectedAppShutdownTimeout,
		},
		Metrics: Metrics{
			Host: expectedMetricsHost,
			Port: expectedMetricsPort,
		},
		Server: Server{
			Host:            expectedServerHost,
			Port:            expectedServerPort,
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
	// OS
	t.Setenv("OS_TZ", expectedOSTimeZone)
	// Application
	t.Setenv("APP_ENV", expectedApplicationEnv)
	t.Setenv("APP_NAME", expectedApplicationName)
	t.Setenv("APP_MODE", expectedApplicationMode)
	t.Setenv("APP_SHUTDOWN_TIMEOUT", expectedAppShutdownTimeoutStr)
	// Server
	t.Setenv("SERVER_HOST", expectedServerHost)
	t.Setenv("SERVER_PORT", strconv.Itoa(expectedServerPort))
	t.Setenv("SERVER_SHUTDOWN_TIMEOUT", expectedServerShutdownTimeoutStr)
	// Metrics
	t.Setenv("METRICS_HOST", expectedMetricsHost)
	t.Setenv("METRICS_PORT", strconv.Itoa(expectedMetricsPort))
	// Database
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
	// Security
	t.Setenv("SECURITY_CIDR", expectedCIDRStr)
	t.Setenv("SECURITY_ALLOWED_ORIGINS", expectedAllowedOrigins)
}
