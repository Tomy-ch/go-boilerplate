package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	// TestingEnvValue は、テスト用の環境変数の値です。
	TestingEnvValue = "ci"
	// testDBHostPort は、共有 DB のホスト公開ポート（テスト接続先・固定）です。
	testDBHostPort = 5432
	// defaultTestDBName は、DB スロットプール未使用時のテスト用データベース既定名です。
	defaultTestDBName = "test"
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
	expectedObservabilityTargetStatusCodes    = []int{400, 401, 403, 404, 405, 409, 422, 429, 500, 501, 503}
	expectedObservabilityTargetStatusCodesStr = "400,401,403,404,405,409,422,429,500,501,503"
	expectedObservabilityTargetStatusCodeSet  = buildStatusCodeSet(expectedObservabilityTargetStatusCodes)
	// database
	expectedDBDriver = "pgx"
	expectedDBHost   = "localhost"
	// worktree の分離はポートではなくデータベース名（DB_NAME_TEST）で行うため、接続ポートは固定。
	expectedDBPort     = testDBHostPort
	expectedDBUser     = "postgres"
	expectedDBPassword = "postgres-password"
	// DB 名は worktree 毎に変わるため testDBName() で解決する（詳細はその doc）。
	expectedDBName                        = testDBName()
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
	// secure cookie
	expectedSecureCookieSecure   = new(true)
	expectedSecureCookieSameSite = "Strict"
	expectedSecureCookieDomain   = "localhost"
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

	// auth（local/ci/test では実 JWT authenticator を配線しないため、issuer 等は既定の空値）
	expectedAuthIssuer             = ""
	expectedAuthAudience           = ""
	expectedAuthJWKSURL            = ""
	expectedAuthAllowedAlgorithms  = []string{"RS256"}
	expectedAuthClockSkew          = 60 * time.Second
	expectedAuthJWKSCacheTTL       = 1 * time.Hour
	expectedAuthDiscoveryTTL       = 24 * time.Hour
	expectedAuthUnknownKidCooldown = 60 * time.Second

	// object storage（Endpoint は空文字＝SDK 既定解決の意味を持つため空。他は required,notEmpty のため実値）
	expectedObjectStorageEndpoint                = ""
	expectedObjectStorageRegion                  = "us-east-1"
	expectedObjectStorageBucket                  = "test-bucket"
	expectedObjectStorageAccessKeyID             = "test-access-key"
	expectedObjectStorageSecretAccessKey         = "test-secret-key"
	expectedObjectStorageUsePathStyle            = true
	expectedObjectStorageUsePathStyleStr         = strconv.FormatBool(expectedObjectStorageUsePathStyle)
	expectedObjectStorageMaxUploadBytes    int64 = 5242880
	expectedObjectStorageMaxUploadBytesStr       = strconv.FormatInt(expectedObjectStorageMaxUploadBytes, 10)
)

// MockConfigForTest は、テスト用のConfigを返します。
//
//nolint:dupl // Config の全フィールドを網羅比較するための構造体リテラルであり、テスト側期待値との重複は不可避
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
		},
		secureCookie: SecureCookieConfig{
			secure:   expectedSecureCookieSecure,
			sameSite: expectedSecureCookieSameSite,
			domain:   expectedSecureCookieDomain,
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
		auth: AuthConfig{
			issuer:             expectedAuthIssuer,
			audience:           expectedAuthAudience,
			jwksURL:            expectedAuthJWKSURL,
			allowedAlgorithms:  expectedAuthAllowedAlgorithms,
			clockSkew:          expectedAuthClockSkew,
			jwksCacheTTL:       expectedAuthJWKSCacheTTL,
			discoveryTTL:       expectedAuthDiscoveryTTL,
			unknownKidCooldown: expectedAuthUnknownKidCooldown,
		},
		objectStorage: ObjectStorageConfig{
			endpoint:        expectedObjectStorageEndpoint,
			region:          expectedObjectStorageRegion,
			bucket:          expectedObjectStorageBucket,
			accessKeyID:     expectedObjectStorageAccessKeyID,
			secretAccessKey: expectedObjectStorageSecretAccessKey,
			usePathStyle:    expectedObjectStorageUsePathStyle,
			maxUploadBytes:  expectedObjectStorageMaxUploadBytes,
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
		},
		SecureCookie: SecureCookie{
			Secure:   expectedSecureCookieSecure,
			SameSite: expectedSecureCookieSameSite,
			Domain:   expectedSecureCookieDomain,
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
	// Secure Cookie
	t.Setenv("SECURE_COOKIE_SECURE", strconv.FormatBool(*expectedSecureCookieSecure))
	t.Setenv("SECURE_COOKIE_SAME_SITE", expectedSecureCookieSameSite)
	t.Setenv("SECURE_COOKIE_DOMAIN", expectedSecureCookieDomain)
	// Auth（make がスロットの issuer を渡すため、実行環境の値が混ざらないよう期待値へ固定する）
	t.Setenv(authIssuerEnvKey, expectedAuthIssuer)
	t.Setenv("AUTH_AUDIENCE", expectedAuthAudience)
	t.Setenv("AUTH_JWKS_URL", expectedAuthJWKSURL)
	// Object Storage
	t.Setenv("OBJECT_STORAGE_ENDPOINT", expectedObjectStorageEndpoint)
	t.Setenv("OBJECT_STORAGE_REGION", expectedObjectStorageRegion)
	t.Setenv("OBJECT_STORAGE_BUCKET", expectedObjectStorageBucket)
	t.Setenv("OBJECT_STORAGE_ACCESS_KEY_ID", expectedObjectStorageAccessKeyID)
	t.Setenv("OBJECT_STORAGE_SECRET_ACCESS_KEY", expectedObjectStorageSecretAccessKey)
	t.Setenv("OBJECT_STORAGE_USE_PATH_STYLE", expectedObjectStorageUsePathStyleStr)
	t.Setenv("OBJECT_STORAGE_MAX_UPLOAD_BYTES", expectedObjectStorageMaxUploadBytesStr)
}

// testDBName は、ホストから見たテスト用データベース名を返します。環境変数 DB_NAME_TEST があればそれを、
// 無ければ既定の "test" を返します。DB スロットプール利用時は make が DB_NAME_TEST を各 worktree の
// テスト用データベース（wt<N>_test）へ設定するため、共有 DB 内の自 worktree DB へ繋ぎます。ただし
// deploy 系 env では本番 DB を誤指しないよう DB_NAME_TEST を無視します（IsLocalClassEnv、未設定は許可）。
func testDBName() string {
	v := os.Getenv("DB_NAME_TEST")
	if v == "" {
		return defaultTestDBName
	}
	if env := os.Getenv("APP_ENV"); env != "" && !IsLocalClassEnv(env) {
		fmt.Fprintf(os.Stderr,
			"[config] 警告: APP_ENV=%q は local/test 系でないため、DB スロットプールの DB_NAME_TEST=%q を無視します\n",
			env, v)
		return defaultTestDBName
	}
	return v
}
