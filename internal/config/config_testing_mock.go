package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	// TestingEnvValue は、テスト用の環境変数の値です。
	TestingEnvValue = "ci"
)

// 下記の変数は、テスト用の期待値以外に、テスト環境の環境変数設定にも使用されます。
// 変更の際は、テストを必ず実行し、環境変数の設定が正しいことを確認してください。
var (
	// operationSystem
	expectedOSTimeZone = "Asia/Tokyo"
	// application
	expectedApplicationEnv      = "test"
	expectedApplicationName     = "TestApp"
	expectedApplicationMode     = DevelopmentMode
	expectedApplicationLogLevel = "debug"
	// SHUTDOWN_TIMEOUT >= REQUEST_TIMEOUT(90s) の交差検証を満たす値にする。
	expectedAppShutdownTimeoutCount = 95
	expectedAppShutdownTimeoutStr   = fmt.Sprintf("%ds", expectedAppShutdownTimeoutCount)
	expectedAppShutdownTimeout      = time.Duration(expectedAppShutdownTimeoutCount) * time.Second
	// server
	expectedServerHost                   = "localhost"
	expectedServerPort                   = 8080
	expectedServerReadHeaderTimeoutCount = 5
	expectedServerReadHeaderTimeoutStr   = fmt.Sprintf("%ds", expectedServerReadHeaderTimeoutCount)
	expectedServerReadHeaderTimeout      = time.Duration(expectedServerReadHeaderTimeoutCount) * time.Second
	expectedServerReadTimeoutCount       = 10
	expectedServerReadTimeoutStr         = fmt.Sprintf("%ds", expectedServerReadTimeoutCount)
	expectedServerReadTimeout            = time.Duration(expectedServerReadTimeoutCount) * time.Second
	expectedServerWriteTimeoutCount      = 95
	expectedServerWriteTimeoutStr        = fmt.Sprintf("%ds", expectedServerWriteTimeoutCount)
	expectedServerWriteTimeout           = time.Duration(expectedServerWriteTimeoutCount) * time.Second
	expectedServerIdleTimeoutCount       = 60
	expectedServerIdleTimeoutStr         = fmt.Sprintf("%ds", expectedServerIdleTimeoutCount)
	expectedServerIdleTimeout            = time.Duration(expectedServerIdleTimeoutCount) * time.Second
	expectedServerBodyLimitMB            = 7
	expectedServerBodyLimitMBStr         = strconv.Itoa(expectedServerBodyLimitMB)
	expectedServerRequestTimeoutCount    = 90
	expectedServerRequestTimeoutStr      = fmt.Sprintf("%ds", expectedServerRequestTimeoutCount)
	expectedServerRequestTimeout         = time.Duration(expectedServerRequestTimeoutCount) * time.Second
	// metrics
	expectedMetricsHost     = "localhost"
	expectedMetricsPort     = 6060
	expectedMetricsUserName = "metrics-user"
	expectedMetricsPassword = "metrics-password"
	// observability
	expectedObservabilityTracesExporter       = "otlp"
	expectedObservabilityMetricsExporter      = "otlp"
	expectedObservabilityLogsExporter         = "otlp"
	expectedObservabilityOTLPEndpoint         = "http://localhost:4318"
	expectedObservabilityOTLPProtocol         = "http/protobuf"
	expectedObservabilityMaskedDBQueryArgs    = false
	expectedObservabilityTargetStatusCodes    = []int{400, 401, 403, 404, 409, 422, 429, 500, 501, 503}
	expectedObservabilityTargetStatusCodesStr = "400,401,403,404,409,422,429,500,501,503"
	expectedObservabilityTargetStatusCodeSet  = buildStatusCodeSet(expectedObservabilityTargetStatusCodes)
	// database
	expectedDBDriver                      = "pgx"
	expectedDBHost                        = "localhost"
	expectedDBPort                        = 5432
	expectedDBUser                        = "postgres"
	expectedDBPassword                    = "postgres-password"
	expectedDBName                        = "test"
	expectedDBSSLMode                     = "disable"
	expectedDBPingTimeoutCount            = 5
	expectedDBPingTimeoutStr              = fmt.Sprintf("%ds", expectedDBPingTimeoutCount)
	expectedDBPingTimeout                 = time.Duration(expectedDBPingTimeoutCount) * time.Second
	expectedDBSlowQueryWarnThresholdCount = 500
	expectedDBSlowQueryWarnThresholdStr   = fmt.Sprintf("%dms", expectedDBSlowQueryWarnThresholdCount)
	expectedDBSlowQueryWarnThreshold      = time.Duration(expectedDBSlowQueryWarnThresholdCount) * time.Millisecond
	expectedDBStatementTimeout            = 30 * time.Second
	expectedDBLockTimeout                 = 10 * time.Second
	expectedDBTxMaxRetries                = 3
	expectedDBTxMaxRetriesStr             = strconv.Itoa(expectedDBTxMaxRetries)
	expectedDBTxRetryBaseBackoffCount     = 5
	expectedDBTxRetryBaseBackoffStr       = fmt.Sprintf("%dms", expectedDBTxRetryBaseBackoffCount)
	expectedDBTxRetryBaseBackoff          = time.Duration(expectedDBTxRetryBaseBackoffCount) * time.Millisecond
	expectedDBTxRetryMaxBackoffCount      = 100
	expectedDBTxRetryMaxBackoffStr        = fmt.Sprintf("%dms", expectedDBTxRetryMaxBackoffCount)
	expectedDBTxRetryMaxBackoff           = time.Duration(expectedDBTxRetryMaxBackoffCount) * time.Millisecond
	// dbconnection
	expectedDBMaxConns         = 10
	expectedDBMaxConnsInt32    = int32(expectedDBMaxConns)
	expectedDBMinConns         = 5
	expectedDBMinConnsInt32    = int32(expectedDBMinConns)
	expectedDBMaxLifetimeCount = 60
	expectedDBMaxLifetimeStr   = fmt.Sprintf("%ds", expectedDBMaxLifetimeCount)
	expectedDBMaxLifetime      = time.Duration(expectedDBMaxLifetimeCount) * time.Second
	expectedDBMaxIdleTimeCount = 30
	expectedDBMaxIdleTimeStr   = fmt.Sprintf("%ds", expectedDBMaxIdleTimeCount)
	expectedDBMaxIdleTime      = time.Duration(expectedDBMaxIdleTimeCount) * time.Second
	// security
	expectedAllowedOrigins        = "http://localhost,https://example.com"
	expectedCIDRStr               = "192.168.0.0/24"
	_, expectedCIDR, _            = net.ParseCIDR(expectedCIDRStr)
	expectedContentTypeNosniff    = "nosniff"
	expectedXFrameOptions         = "SAMEORIGIN"
	expectedHSTSMaxAgeCount       = 31536000
	expectedHSTSMaxAge            = time.Duration(expectedHSTSMaxAgeCount) * time.Second
	expectedHSTSMaxAgeStr         = fmt.Sprintf("%ds", expectedHSTSMaxAgeCount)
	expectedHSTSExcludeSubdomains = false
	expectedHSTSPreloadEnabled    = false
	expectedReferrerPolicy        = "no-referrer"
	expectedBcryptCost            = bcrypt.MinCost
	// secure cookie
	expectedSecureCookieSecure   = new(true)
	expectedSecureCookieSameSite = "Strict"
	expectedSecureCookieDomain   = "localhost"
	// auth
	expectedAuthCookieName          = "auth_token"
	expectedAuthHeaderName          = "Authorization"
	expectedAuthAllowedHeaderBearer = true
	// worker
	expectedWorkerConcurrency               = 4
	expectedWorkerMaxInFlight               = 8
	expectedWorkerBatchSize                 = 4
	expectedWorkerExtendInterval            = time.Duration(0)
	expectedWorkerDrainTimeout              = 30 * time.Second
	expectedWorkerReceiveCountWarnThreshold = 5
	expectedWorkerCircuitFailureThreshold   = 10
	expectedWorkerCircuitOpenBackoffInitial = 1 * time.Second
	expectedWorkerCircuitOpenBackoffMax     = 30 * time.Second
	expectedWorkerCircuitHalfOpenProbe      = 1
	expectedWorkerHealthListenAddr          = ":8081"
	expectedWorkerProgressStaleAfter        = 60 * time.Second
	expectedWorkerNackBackoffInitial        = 1 * time.Second
	expectedWorkerNackBackoffMax            = 30 * time.Second

	// outbox
	expectedOutboxEndpoint     = ""
	expectedOutboxPollInterval = 1 * time.Second
	expectedOutboxErrorBackoff = 5 * time.Second
	expectedOutboxBatchSize    = 100
)

// MockConfigForTest は、テスト用のConfigを返します。
func MockConfigForTest(tb testing.TB) *Config {
	tb.Helper()
	return &Config{
		os: OperatingSystemConfig{
			timezone: expectedOSTimeZone,
		},
		app: ApplicationConfig{
			env:             expectedApplicationEnv,
			name:            expectedApplicationName,
			mode:            expectedApplicationMode,
			logLevel:        expectedApplicationLogLevel,
			shutdownTimeout: expectedAppShutdownTimeout,
		},
		server: ServerConfig{
			host:              expectedServerHost,
			port:              expectedServerPort,
			readHeaderTimeout: expectedServerReadHeaderTimeout,
			readTimeout:       expectedServerReadTimeout,
			writeTimeout:      expectedServerWriteTimeout,
			idleTimeout:       expectedServerIdleTimeout,
			bodyLimitMB:       expectedServerBodyLimitMB,
			requestTimeout:    expectedServerRequestTimeout,
		},
		metrics: MetricsConfig{
			host:     expectedMetricsHost,
			port:     expectedMetricsPort,
			userName: expectedMetricsUserName,
			password: expectedMetricsPassword,
		},
		observability: ObservabilityConfig{
			tracesExporter:      expectedObservabilityTracesExporter,
			metricsExporter:     expectedObservabilityMetricsExporter,
			logsExporter:        expectedObservabilityLogsExporter,
			otlpEndpoint:        expectedObservabilityOTLPEndpoint,
			otlpProtocol:        expectedObservabilityOTLPProtocol,
			maskedDBQueryArgs:   expectedObservabilityMaskedDBQueryArgs,
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
			statementTimeout:       expectedDBStatementTimeout,
			lockTimeout:            expectedDBLockTimeout,
			txMaxRetries:           expectedDBTxMaxRetries,
			txRetryBaseBackoff:     expectedDBTxRetryBaseBackoff,
			txRetryMaxBackoff:      expectedDBTxRetryMaxBackoff,
		},
		dbconnection: DBConnectionConfig{
			maxConns:    expectedDBMaxConnsInt32,
			minConns:    expectedDBMinConnsInt32,
			maxLifetime: expectedDBMaxLifetime,
			maxIdleTime: expectedDBMaxIdleTime,
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
		worker: WorkerConfig{
			concurrency:               expectedWorkerConcurrency,
			maxInFlight:               expectedWorkerMaxInFlight,
			batchSize:                 expectedWorkerBatchSize,
			extendInterval:            expectedWorkerExtendInterval,
			drainTimeout:              expectedWorkerDrainTimeout,
			receiveCountWarnThreshold: expectedWorkerReceiveCountWarnThreshold,
			circuitFailureThreshold:   expectedWorkerCircuitFailureThreshold,
			circuitOpenBackoffInitial: expectedWorkerCircuitOpenBackoffInitial,
			circuitOpenBackoffMax:     expectedWorkerCircuitOpenBackoffMax,
			circuitHalfOpenProbe:      expectedWorkerCircuitHalfOpenProbe,
			healthListenAddr:          expectedWorkerHealthListenAddr,
			progressStaleAfter:        expectedWorkerProgressStaleAfter,
			nackBackoffInitial:        expectedWorkerNackBackoffInitial,
			nackBackoffMax:            expectedWorkerNackBackoffMax,
		},
		outbox: OutboxConfig{
			endpoint:     expectedOutboxEndpoint,
			pollInterval: expectedOutboxPollInterval,
			errorBackoff: expectedOutboxErrorBackoff,
			batchSize:    expectedOutboxBatchSize,
		},
	}
}

// mockLoader は、テスト用のLoaderを返します。
func mockLoader(tb testing.TB) Loader {
	tb.Helper()

	return Loader{
		OS: OperatingSystem{
			Timezone: expectedOSTimeZone,
		},
		App: Application{
			Env:             expectedApplicationEnv,
			Name:            expectedApplicationName,
			Mode:            expectedApplicationMode,
			LogLevel:        expectedApplicationLogLevel,
			ShutdownTimeout: expectedAppShutdownTimeout,
		},
		Metrics: Metrics{
			Host:     expectedMetricsHost,
			Port:     expectedMetricsPort,
			UserName: expectedMetricsUserName,
			Password: expectedMetricsPassword,
		},
		Observability: Observability{
			TracesExporter:    expectedObservabilityTracesExporter,
			MetricsExporter:   expectedObservabilityMetricsExporter,
			LogsExporter:      expectedObservabilityLogsExporter,
			OTLPEndpoint:      expectedObservabilityOTLPEndpoint,
			OTLPProtocol:      expectedObservabilityOTLPProtocol,
			MaskedDBQueryArgs: expectedObservabilityMaskedDBQueryArgs,
			TargetStatusCodes: expectedObservabilityTargetStatusCodes,
		},
		Server: Server{
			Host:              expectedServerHost,
			Port:              expectedServerPort,
			ReadHeaderTimeout: expectedServerReadHeaderTimeout,
			ReadTimeout:       expectedServerReadTimeout,
			WriteTimeout:      expectedServerWriteTimeout,
			IdleTimeout:       expectedServerIdleTimeout,
			BodyLimitMB:       expectedServerBodyLimitMB,
			RequestTimeout:    expectedServerRequestTimeout,
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
			StatementTimeout:       expectedDBStatementTimeout,
			LockTimeout:            expectedDBLockTimeout,
			TxMaxRetries:           expectedDBTxMaxRetries,
			TxRetryBaseBackoff:     expectedDBTxRetryBaseBackoff,
			TxRetryMaxBackoff:      expectedDBTxRetryMaxBackoff,
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
	}
}

func setEnvVarsForTesting(t *testing.T) { //nolint:funlen // テスト用の環境変数設定のため長くなるのは許容する
	t.Helper()
	// OS
	t.Setenv("OS_TZ", expectedOSTimeZone)
	// Application
	t.Setenv("APP_ENV", expectedApplicationEnv)
	t.Setenv("APP_NAME", expectedApplicationName)
	t.Setenv("APP_MODE", expectedApplicationMode)
	t.Setenv("APP_LOG_LEVEL", expectedApplicationLogLevel)
	t.Setenv("APP_SHUTDOWN_TIMEOUT", expectedAppShutdownTimeoutStr)
	// Server
	t.Setenv("SERVER_HOST", expectedServerHost)
	t.Setenv("SERVER_PORT", strconv.Itoa(expectedServerPort))
	t.Setenv("SERVER_READ_HEADER_TIMEOUT", expectedServerReadHeaderTimeoutStr)
	t.Setenv("SERVER_READ_TIMEOUT", expectedServerReadTimeoutStr)
	t.Setenv("SERVER_WRITE_TIMEOUT", expectedServerWriteTimeoutStr)
	t.Setenv("SERVER_IDLE_TIMEOUT", expectedServerIdleTimeoutStr)
	t.Setenv("SERVER_BODY_LIMIT_MB", expectedServerBodyLimitMBStr)
	t.Setenv("SERVER_REQUEST_TIMEOUT", expectedServerRequestTimeoutStr)
	// Metrics
	t.Setenv("METRICS_HOST", expectedMetricsHost)
	t.Setenv("METRICS_PORT", strconv.Itoa(expectedMetricsPort))
	t.Setenv("METRICS_USERNAME", expectedMetricsUserName)
	t.Setenv("METRICS_PASSWORD", expectedMetricsPassword)
	// Observability
	t.Setenv("OBS_TRACES_EXPORTER", expectedObservabilityTracesExporter)
	t.Setenv("OBS_METRICS_EXPORTER", expectedObservabilityMetricsExporter)
	t.Setenv("OBS_LOGS_EXPORTER", expectedObservabilityLogsExporter)
	t.Setenv("OBS_OTLP_ENDPOINT", expectedObservabilityOTLPEndpoint)
	t.Setenv("OBS_OTLP_PROTOCOL", expectedObservabilityOTLPProtocol)
	t.Setenv("OBS_MASKED_DB_QUERY_ARGS", strconv.FormatBool(expectedObservabilityMaskedDBQueryArgs))
	t.Setenv("OBS_TARGET_STATUS_CODES", expectedObservabilityTargetStatusCodesStr)
	// Database
	t.Setenv("DB_DRIVER", expectedDBDriver)
	t.Setenv("DB_HOST", expectedDBHost)
	t.Setenv("DB_PORT", strconv.Itoa(expectedDBPort))
	t.Setenv("DB_USER", expectedDBUser)
	t.Setenv("DB_PASSWORD", expectedDBPassword)
	t.Setenv("DB_NAME", expectedDBName)
	t.Setenv("DB_SSL_MODE", expectedDBSSLMode)
	t.Setenv("DB_PING_TIMEOUT", expectedDBPingTimeoutStr)
	t.Setenv("DB_SLOW_QUERY_WARN_THRESHOLD", expectedDBSlowQueryWarnThresholdStr)
	t.Setenv("DB_STATEMENT_TIMEOUT", expectedDBStatementTimeout.String())
	t.Setenv("DB_LOCK_TIMEOUT", expectedDBLockTimeout.String())
	t.Setenv("DB_TX_MAX_RETRIES", expectedDBTxMaxRetriesStr)
	t.Setenv("DB_TX_RETRY_BASE_BACKOFF", expectedDBTxRetryBaseBackoffStr)
	t.Setenv("DB_TX_RETRY_MAX_BACKOFF", expectedDBTxRetryMaxBackoffStr)
	// DBConnection
	t.Setenv("DBCONN_MAX_CONNS", strconv.FormatInt(int64(expectedDBMaxConnsInt32), 10))
	t.Setenv("DBCONN_MIN_CONNS", strconv.FormatInt(int64(expectedDBMinConnsInt32), 10))
	t.Setenv("DBCONN_MAX_LIFETIME", expectedDBMaxLifetimeStr)
	t.Setenv("DBCONN_MAX_IDLE_TIME", expectedDBMaxIdleTimeStr)
	// Security
	t.Setenv("SECURITY_CIDR", expectedCIDRStr)
	t.Setenv("SECURITY_ALLOWED_ORIGINS", expectedAllowedOrigins)
	t.Setenv("SECURITY_CONTENT_TYPE_NOSNIFF", expectedContentTypeNosniff)
	t.Setenv("SECURITY_X_FRAME_OPTIONS", expectedXFrameOptions)
	t.Setenv("SECURITY_HSTS_MAX_AGE", expectedHSTSMaxAgeStr)
	t.Setenv("SECURITY_HSTS_EXCLUDE_SUBDOMAINS", strconv.FormatBool(expectedHSTSExcludeSubdomains))
	t.Setenv("SECURITY_HSTS_PRELOAD_ENABLED", strconv.FormatBool(expectedHSTSPreloadEnabled))
	t.Setenv("SECURITY_REFERRER_POLICY", expectedReferrerPolicy)
	t.Setenv("SECURITY_BCRYPT_COST", strconv.Itoa(expectedBcryptCost))
	// Secure Cookie
	t.Setenv("SECURE_COOKIE_SECURE", strconv.FormatBool(*expectedSecureCookieSecure))
	t.Setenv("SECURE_COOKIE_SAME_SITE", expectedSecureCookieSameSite)
	t.Setenv("SECURE_COOKIE_DOMAIN", expectedSecureCookieDomain)
	// Auth
	t.Setenv("AUTH_COOKIE_NAME", expectedAuthCookieName)
	t.Setenv("AUTH_HEADER_NAME", expectedAuthHeaderName)
	t.Setenv("AUTH_ALLOWED_HEADER_BEARER", strconv.FormatBool(expectedAuthAllowedHeaderBearer))
}
