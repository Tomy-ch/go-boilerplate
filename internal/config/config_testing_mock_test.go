package config

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockConfigForTest(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("MockConfigForTest は、テスト用の Config を返す", func(t *testing.T) {
			t.Parallel()
			expected := &Config{
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
					issuer:            expectedAuthIssuer,
					audience:          expectedAuthAudience,
					jwksURL:           expectedAuthJWKSURL,
					allowedAlgorithms: expectedAuthAllowedAlgorithms,
					clockSkew:         expectedAuthClockSkew,
					jwksCacheTTL:      expectedAuthJWKSCacheTTL,
				},
			}

			actual := MockConfigForTest(t)

			assert.Equal(t, expected, actual)
		})
	})
}

func Test_mockLoader(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("mockLoader は、テスト用の Loader を返す", func(t *testing.T) {
			t.Parallel()
			expected := Loader{
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
			}

			actual := mockLoader(t)

			assert.Equal(t, expected, actual)
		})
	})
}

func Test_setEnv(t *testing.T) { //nolint:paralleltest // t.Setenv/t.Chdir使用のため並列化不可
	t.Run("正常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv/t.Chdir使用のため並列化不可
		t.Run("setEnvVarsForTesting の環境変数設定は、テスト内で正しく反映される", func(t *testing.T) { //nolint:paralleltest // t.Setenv/t.Chdir使用のため並列化不可
			// setEnvVarsForTesting が t.Setenv を使うため、本テストは t.Parallel() を付けられない。
			setEnvVarsForTesting(t)
			// OS
			assert.Equal(t, expectedOSTimeZone, os.Getenv("OS_TZ"))
			// Application
			assert.Equal(t, expectedApplicationEnv, os.Getenv("APP_ENV"))
			assert.Equal(t, expectedApplicationName, os.Getenv("APP_NAME"))
			assert.Equal(t, expectedApplicationMode, os.Getenv("APP_MODE"))
			assert.Equal(t, expectedApplicationLogLevel, os.Getenv("APP_LOG_LEVEL"))
			assert.Equal(t, expectedAppShutdownTimeoutStr, os.Getenv("APP_SHUTDOWN_TIMEOUT"))
			// Server
			assert.Equal(t, expectedServerHost, os.Getenv("SERVER_HOST"))
			assert.Equal(t, strconv.Itoa(expectedServerPort), os.Getenv("SERVER_PORT"))
			assert.Equal(t, expectedServerReadHeaderTimeoutStr, os.Getenv("SERVER_READ_HEADER_TIMEOUT"))
			assert.Equal(t, expectedServerReadTimeoutStr, os.Getenv("SERVER_READ_TIMEOUT"))
			assert.Equal(t, expectedServerWriteTimeoutStr, os.Getenv("SERVER_WRITE_TIMEOUT"))
			assert.Equal(t, expectedServerIdleTimeoutStr, os.Getenv("SERVER_IDLE_TIMEOUT"))
			// Metrics
			assert.Equal(t, expectedMetricsHost, os.Getenv("METRICS_HOST"))
			assert.Equal(t, strconv.Itoa(expectedMetricsPort), os.Getenv("METRICS_PORT"))
			assert.Equal(t, expectedMetricsUserName, os.Getenv("METRICS_USERNAME"))
			assert.Equal(t, expectedMetricsPassword, os.Getenv("METRICS_PASSWORD"))
			// Observability
			assert.Equal(t, expectedObservabilityTracesExporter, os.Getenv("OBS_TRACES_EXPORTER"))
			assert.Equal(t, expectedObservabilityMetricsExporter, os.Getenv("OBS_METRICS_EXPORTER"))
			assert.Equal(t, expectedObservabilityLogsExporter, os.Getenv("OBS_LOGS_EXPORTER"))
			assert.Equal(t, expectedObservabilityOTLPEndpoint, os.Getenv("OBS_OTLP_ENDPOINT"))
			assert.Equal(t, expectedObservabilityOTLPProtocol, os.Getenv("OBS_OTLP_PROTOCOL"))
			assert.Equal(t, strconv.FormatBool(expectedObservabilityMaskedDBQueryArgs), os.Getenv("OBS_MASKED_DB_QUERY_ARGS"))
			assert.Equal(t, expectedObservabilityTargetStatusCodesStr, os.Getenv("OBS_TARGET_STATUS_CODES"))
			// Database
			assert.Equal(t, expectedDBDriver, os.Getenv("DB_DRIVER"))
			assert.Equal(t, expectedDBHost, os.Getenv("DB_HOST"))
			assert.Equal(t, strconv.Itoa(expectedDBPort), os.Getenv("DB_PORT"))
			assert.Equal(t, expectedDBUser, os.Getenv("DB_USER"))
			assert.Equal(t, expectedDBPassword, os.Getenv("DB_PASSWORD"))
			assert.Equal(t, expectedDBName, os.Getenv("DB_NAME"))
			assert.Equal(t, expectedDBSSLMode, os.Getenv("DB_SSL_MODE"))
			assert.Equal(t, expectedDBPingTimeoutStr, os.Getenv("DB_PING_TIMEOUT"))
			assert.Equal(t, expectedDBSlowQueryWarnThresholdStr, os.Getenv("DB_SLOW_QUERY_WARN_THRESHOLD"))
			// DBConnection
			assert.Equal(t, strconv.FormatInt(int64(expectedDBMaxConnsInt32), 10), os.Getenv("DBCONN_MAX_CONNS"))
			assert.Equal(t, strconv.FormatInt(int64(expectedDBMinConnsInt32), 10), os.Getenv("DBCONN_MIN_CONNS"))
			assert.Equal(t, expectedDBMaxLifetimeStr, os.Getenv("DBCONN_MAX_LIFETIME"))
			assert.Equal(t, expectedDBMaxIdleTimeStr, os.Getenv("DBCONN_MAX_IDLE_TIME"))
			// Security
			assert.Equal(t, expectedCIDRStr, os.Getenv("SECURITY_CIDR"))
			assert.Equal(t, expectedAllowedOrigins, os.Getenv("SECURITY_ALLOWED_ORIGINS"))
			assert.Equal(t, expectedContentTypeNosniff, os.Getenv("SECURITY_CONTENT_TYPE_NOSNIFF"))
			assert.Equal(t, expectedXFrameOptions, os.Getenv("SECURITY_X_FRAME_OPTIONS"))
			assert.Equal(t, expectedHSTSMaxAgeStr, os.Getenv("SECURITY_HSTS_MAX_AGE"))
			assert.Equal(t, strconv.FormatBool(expectedHSTSExcludeSubdomains), os.Getenv("SECURITY_HSTS_EXCLUDE_SUBDOMAINS"))
			assert.Equal(t, strconv.FormatBool(expectedHSTSPreloadEnabled), os.Getenv("SECURITY_HSTS_PRELOAD_ENABLED"))
			assert.Equal(t, expectedReferrerPolicy, os.Getenv("SECURITY_REFERRER_POLICY"))
			assert.Equal(t, strconv.Itoa(expectedBcryptCost), os.Getenv("SECURITY_BCRYPT_COST"))
			// Secure Cookie
			assert.Equal(t, strconv.FormatBool(*expectedSecureCookieSecure), os.Getenv("SECURE_COOKIE_SECURE"))
			assert.Equal(t, expectedSecureCookieSameSite, os.Getenv("SECURE_COOKIE_SAME_SITE"))
			assert.Equal(t, expectedSecureCookieDomain, os.Getenv("SECURE_COOKIE_DOMAIN"))
		})
	})
}
