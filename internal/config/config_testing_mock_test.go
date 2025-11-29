package config

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMockConfigForTest(t *testing.T) {
	t.Parallel()
	expected := &Config{
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

	actual := MockConfigForTest(t)

	require.Equal(t, expected, actual)
}

func Test_mockLoader(t *testing.T) {
	t.Parallel()
	expected := Loader{
		OS: OperationSystem{
			Timezone: expectedOSTimeZone,
		},
		App: Application{
			Env:             expectedApplicationEnv,
			Name:            expectedApplicationName,
			Mode:            expectedApplicationMode,
			ShutdownTimeout: expectedAppShutdownTimeout,
		},
		Server: Server{
			Host:            expectedServerHost,
			Port:            expectedServerPort,
			ShutdownTimeout: expectedServerShutdownTimeout,
		},
		Metrics: Metrics{
			Host: expectedMetricsHost,
			Port: expectedMetricsPort,
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

	actual := mockLoader(t)

	require.Equal(t, expected, actual)
}

func Test_setEnv(t *testing.T) {
	setEnv(t)
	// OS
	require.Equal(t, expectedOSTimeZone, os.Getenv("OS_TZ"))
	// Application
	require.Equal(t, expectedApplicationEnv, os.Getenv("APP_ENV"))
	require.Equal(t, expectedApplicationMode, os.Getenv("APP_MODE"))
	require.Equal(t, expectedAppShutdownTimeoutStr, os.Getenv("APP_SHUTDOWN_TIMEOUT"))
	// Server
	require.Equal(t, expectedServerHost, os.Getenv("SERVER_HOST"))
	require.Equal(t, strconv.Itoa(expectedServerPort), os.Getenv("SERVER_PORT"))
	require.Equal(t, expectedServerShutdownTimeoutStr, os.Getenv("SERVER_SHUTDOWN_TIMEOUT"))
	// Metrics
	require.Equal(t, expectedMetricsHost, os.Getenv("METRICS_HOST"))
	require.Equal(t, strconv.Itoa(expectedMetricsPort), os.Getenv("METRICS_PORT"))
	// Database
	require.Equal(t, expectedDBDriver, os.Getenv("DB_DRIVER"))
	require.Equal(t, expectedDBHost, os.Getenv("DB_HOST"))
	require.Equal(t, strconv.Itoa(expectedDBPort), os.Getenv("DB_PORT"))
	require.Equal(t, expectedDBUser, os.Getenv("DB_USER"))
	require.Equal(t, expectedDBPassword, os.Getenv("DB_PASSWORD"))
	require.Equal(t, expectedDBName, os.Getenv("DB_NAME"))
	require.Equal(t, expectedDBSSLMode, os.Getenv("DB_SSL_MODE"))
	require.Equal(t, strconv.Itoa(expectedDBMaxOpenConns), os.Getenv("DB_CONN_MAX_OPEN"))
	require.Equal(t, strconv.Itoa(expectedDBMaxIdleConns), os.Getenv("DB_CONN_MAX_IDLE"))
	require.Equal(t, expectedDBMaxLifetimeStr, os.Getenv("DB_CONN_MAX_LIFETIME"))
	require.Equal(t, expectedDBMaxIdleTimeStr, os.Getenv("DB_CONN_MAX_IDLE_TIME"))
	// Security
	require.Equal(t, expectedCIDRStr, os.Getenv("SECURITY_CIDR"))
	require.Equal(t, expectedAllowedOrigins, os.Getenv("SECURITY_ALLOWED_ORIGINS"))
}
