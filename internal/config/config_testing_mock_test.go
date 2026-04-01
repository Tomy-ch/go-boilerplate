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
			host:              expectedServerHost,
			port:              expectedServerPort,
			readHeaderTimeout: expectedServerReadHeaderTimeout,
			readTimeout:       expectedServerReadTimeout,
			writeTimeout:      expectedServerWriteTimeout,
			idleTimeout:       expectedServerIdleTimeout,
		},
		metrics: MetricsConfig{
			host:     expectedMetricsHost,
			port:     expectedMetricsPort,
			userName: expectedMetricsUserName,
			password: expectedMetricsPassword,
		},
		observability: ObservabilityConfig{
			enabled:             expectedObservabilityEnabled,
			maskedDBQueryArgs:   expectedObservabilityMaskedDBQueryArgs,
			targetStatusCodes:   expectedObservabilityTargetStatusCodes,
			targetStatusCodeSet: expectedObservabilityTargetStatusCodeSet,
		},
		database: DatabaseConfig{
			driver:                 expectedDBDriver,
			host:                   expectedDBHost,
			port:                   expectedDBPort,
			user:                   expectedDBUser,
			password:               expectedDBPassword,
			name:                   expectedDBName,
			sslMode:                expectedDBSSLMode,
			pingTimeout:            expectedDBPingTimeout,
			slowQueryWarnThreshold: expectedDBSlowQueryWarnThreshold,
		},
		dbconnection: DBConnectionConfig{
			maxOpenConns: expectedDBMaxConnsInt32,
			maxIdleConns: expectedDBMinConnsInt32,
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
			bcryptCost:            expectedBcryptCost,
		},
		secureCookie: SecureCookieConfig{
			secure:   expectedSecureCookieSecure,
			sameSite: expectedSecureCookieSameSite,
			domain:   expectedSecureCookieDomain,
		},
		auth: AuthConfig{
			cookieName:          expectedAuthCookieName,
			headerName:          expectedAuthHeaderName,
			allowedHeaderBearer: expectedAuthAllowedHeaderBearer,
		},
		ipRateLimit: IPRateLimitConfig{
			enabled:         expectedIPRateLimitEnabled,
			requests:        expectedIPRateLimitRequests,
			per:             expectedIPRateLimitPer,
			burst:           expectedIPRateLimitBurst,
			ttl:             expectedIPRateLimitTTL,
			cleanupInterval: expectedIPRateLimitCleanupInterval,
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
			Host:              expectedServerHost,
			Port:              expectedServerPort,
			ReadHeaderTimeout: expectedServerReadHeaderTimeout,
			ReadTimeout:       expectedServerReadTimeout,
			WriteTimeout:      expectedServerWriteTimeout,
			IdleTimeout:       expectedServerIdleTimeout,
		},
		Metrics: Metrics{
			Host:     expectedMetricsHost,
			Port:     expectedMetricsPort,
			UserName: expectedMetricsUserName,
			Password: expectedMetricsPassword,
		},
		Observability: Observability{
			Enabled:           expectedObservabilityEnabled,
			MaskedDBQueryArgs: expectedObservabilityMaskedDBQueryArgs,
			TargetStatusCodes: expectedObservabilityTargetStatusCodes,
		},
		Database: Database{
			Host:                   expectedDBHost,
			Port:                   expectedDBPort,
			User:                   expectedDBUser,
			Password:               expectedDBPassword,
			Name:                   expectedDBName,
			SSLMode:                expectedDBSSLMode,
			PingTimeout:            expectedDBPingTimeout,
			SlowQueryWarnThreshold: expectedDBSlowQueryWarnThreshold,
		},
		DBConnection: DBConnection{
			MaxConns:    expectedDBMaxConnsInt32,
			MinConns:    expectedDBMinConnsInt32,
			MaxLifetime: expectedDBMaxLifetime,
			MaxIdleTime: expectedDBMaxIdleTime,
		},
		Security: Security{
			AllowedOrigins:        strings.Split(expectedAllowedOrigins, ","),
			CIDR:                  expectedCIDRStr,
			ContentTypeNosniff:    expectedContentTypeNosniff,
			XFrameOptions:         expectedXFrameOptions,
			HSTSMaxAge:            expectedHSTSMaxAge,
			HSTSExcludeSubdomains: expectedHSTSExcludeSubdomains,
			HSTSPreloadEnabled:    expectedHSTSPreloadEnabled,
			ReferrerPolicy:        expectedReferrerPolicy,
			BcryptCost:            expectedBcryptCost,
		},
		SecureCookie: SecureCookie{
			Secure:   expectedSecureCookieSecure,
			SameSite: expectedSecureCookieSameSite,
			Domain:   expectedSecureCookieDomain,
		},
		Auth: Auth{
			CookieName:          expectedAuthCookieName,
			HeaderName:          expectedAuthHeaderName,
			AllowedHeaderBearer: expectedAuthAllowedHeaderBearer,
		},
		IPRateLimit: IPRateLimit{
			Enabled:         expectedIPRateLimitEnabled,
			Requests:        expectedIPRateLimitRequests,
			Per:             expectedIPRateLimitPer,
			Burst:           expectedIPRateLimitBurst,
			TTL:             expectedIPRateLimitTTL,
			CleanupInterval: expectedIPRateLimitCleanupInterval,
		},
	}

	actual := mockLoader(t)

	require.Equal(t, expected, actual)
}

func Test_setEnv(t *testing.T) { //nolint:funlen // safe: This function is only used
	//  for testing and setting environment variables, so the complexity is acceptable.
	setEnvVarsForTesting(t)
	// OS
	require.Equal(t, expectedOSTimeZone, os.Getenv("OS_TZ"))
	// Application
	require.Equal(t, expectedApplicationEnv, os.Getenv("APP_ENV"))
	require.Equal(t, expectedApplicationMode, os.Getenv("APP_MODE"))
	require.Equal(t, expectedAppShutdownTimeoutStr, os.Getenv("APP_SHUTDOWN_TIMEOUT"))
	// Server
	require.Equal(t, expectedServerHost, os.Getenv("SERVER_HOST"))
	require.Equal(t, strconv.Itoa(expectedServerPort), os.Getenv("SERVER_PORT"))
	require.Equal(t, expectedServerReadHeaderTimeoutStr, os.Getenv("SERVER_READ_HEADER_TIMEOUT"))
	require.Equal(t, expectedServerReadTimeoutStr, os.Getenv("SERVER_READ_TIMEOUT"))
	require.Equal(t, expectedServerWriteTimeoutStr, os.Getenv("SERVER_WRITE_TIMEOUT"))
	require.Equal(t, expectedServerIdleTimeoutStr, os.Getenv("SERVER_IDLE_TIMEOUT"))
	// Metrics
	require.Equal(t, expectedMetricsHost, os.Getenv("METRICS_HOST"))
	require.Equal(t, strconv.Itoa(expectedMetricsPort), os.Getenv("METRICS_PORT"))
	require.Equal(t, expectedMetricsUserName, os.Getenv("METRICS_USERNAME"))
	require.Equal(t, expectedMetricsPassword, os.Getenv("METRICS_PASSWORD"))
	// Observability
	require.Equal(t, strconv.FormatBool(expectedObservabilityEnabled), os.Getenv("OBSERVABILITY_ENABLED"))
	require.Equal(t, strconv.FormatBool(expectedObservabilityMaskedDBQueryArgs), os.Getenv("OBSERVABILITY_MASKED_DB_QUERY_ARGS"))
	require.Equal(t, expectedObservabilityTargetStatusCodesStr, os.Getenv("OBSERVABILITY_TARGET_STATUS_CODES"))
	// Database
	require.Equal(t, expectedDBDriver, os.Getenv("DB_DRIVER"))
	require.Equal(t, expectedDBHost, os.Getenv("DB_HOST"))
	require.Equal(t, strconv.Itoa(expectedDBPort), os.Getenv("DB_PORT"))
	require.Equal(t, expectedDBUser, os.Getenv("DB_USER"))
	require.Equal(t, expectedDBPassword, os.Getenv("DB_PASSWORD"))
	require.Equal(t, expectedDBName, os.Getenv("DB_NAME"))
	require.Equal(t, expectedDBSSLMode, os.Getenv("DB_SSL_MODE"))
	require.Equal(t, expectedDBPingTimeoutStr, os.Getenv("DB_PING_TIMEOUT"))
	require.Equal(t, expectedDBSlowQueryWarnThresholdStr, os.Getenv("DB_SLOW_QUERY_WARN_THRESHOLD"))
	// DBConnection
	require.Equal(t, strconv.FormatInt(int64(expectedDBMaxConnsInt32), 10), os.Getenv("DBCONN_MAX_CONNS"))
	require.Equal(t, strconv.FormatInt(int64(expectedDBMinConnsInt32), 10), os.Getenv("DBCONN_MIN_CONNS"))
	require.Equal(t, expectedDBMaxLifetimeStr, os.Getenv("DBCONN_MAX_LIFETIME"))
	require.Equal(t, expectedDBMaxIdleTimeStr, os.Getenv("DBCONN_MAX_IDLE_TIME"))
	// Security
	require.Equal(t, expectedCIDRStr, os.Getenv("SECURITY_CIDR"))
	require.Equal(t, expectedAllowedOrigins, os.Getenv("SECURITY_ALLOWED_ORIGINS"))
	require.Equal(t, expectedContentTypeNosniff, os.Getenv("SECURITY_CONTENT_TYPE_NOSNIFF"))
	require.Equal(t, expectedXFrameOptions, os.Getenv("SECURITY_X_FRAME_OPTIONS"))
	require.Equal(t, expectedHSTSMaxAgeStr, os.Getenv("SECURITY_HSTS_MAX_AGE"))
	require.Equal(t, strconv.FormatBool(expectedHSTSExcludeSubdomains), os.Getenv("SECURITY_HSTS_EXCLUDE_SUBDOMAINS"))
	require.Equal(t, strconv.FormatBool(expectedHSTSPreloadEnabled), os.Getenv("SECURITY_HSTS_PRELOAD_ENABLED"))
	require.Equal(t, expectedReferrerPolicy, os.Getenv("SECURITY_REFERRER_POLICY"))
	require.Equal(t, strconv.Itoa(expectedBcryptCost), os.Getenv("SECURITY_BCRYPT_COST"))
	// Secure Cookie
	require.Equal(t, strconv.FormatBool(*expectedSecureCookieSecure), os.Getenv("SECURE_COOKIE_SECURE"))
	require.Equal(t, expectedSecureCookieSameSite, os.Getenv("SECURE_COOKIE_SAME_SITE"))
	require.Equal(t, expectedSecureCookieDomain, os.Getenv("SECURE_COOKIE_DOMAIN"))
	// Auth
	require.Equal(t, expectedAuthCookieName, os.Getenv("AUTH_COOKIE_NAME"))
	require.Equal(t, expectedAuthHeaderName, os.Getenv("AUTH_HEADER_NAME"))
	require.Equal(t, strconv.FormatBool(expectedAuthAllowedHeaderBearer), os.Getenv("AUTH_ALLOWED_HEADER_BEARER"))
	// IPRateLimit
	require.Equal(t, strconv.FormatBool(expectedIPRateLimitEnabled), os.Getenv("IP_RATE_LIMITER_ENABLED"))
	require.Equal(t, strconv.Itoa(expectedIPRateLimitRequests), os.Getenv("IP_RATE_LIMITER_REQUESTS"))
	require.Equal(t, expectedIPRateLimitPerStr, os.Getenv("IP_RATE_LIMITER_PER"))
	require.Equal(t, strconv.Itoa(expectedIPRateLimitBurst), os.Getenv("IP_RATE_LIMITER_BURST"))
	require.Equal(t, expectedIPRateLimitTTLStr, os.Getenv("IP_RATE_LIMITER_TTL"))
	require.Equal(t, expectedIPRateLimitCleanupIntervalStr, os.Getenv("IP_RATE_LIMITER_CLEANUP_INTERVAL"))
}
