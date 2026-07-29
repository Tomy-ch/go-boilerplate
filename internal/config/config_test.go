package config

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("正常系", func(t *testing.T) { //nolint:paralleltest // t.Setenv/t.Chdir使用のため並列化不可
		t.Run("configに必要な環境変数が全て設定されている場合", func(t *testing.T) { //nolint:paralleltest // t.Setenv/t.Chdir使用のため並列化不可
			setEnvVarsForTesting(t)
			//nolint:dupl // Config の全フィールドを網羅比較するための構造体リテラルであり、mock 側期待値との重複は不可避
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

			actual, err := New()

			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("configに必要な環境変数が不足している場合", func(t *testing.T) {
			t.Setenv("SERVER_ENV", expectedApplicationEnv)

			actual, err := New()
			assert.Nil(t, actual)
			require.ErrorContains(t, err, "APP_MODE")
		})

		t.Run("バリデート結果がエラーの場合", func(t *testing.T) {
			setEnvVarsForTesting(t)
			t.Setenv("APP_MODE", "invalid_env")

			actual, err := New()
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidAppMode)
		})

		t.Run("CIDRのパースに失敗した場合", func(t *testing.T) {
			setEnvVarsForTesting(t)
			t.Setenv("SECURITY_CIDR", "invalid_cidr") // 無効なCIDR

			actual, err := New()
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrFailedToParseCIDR)
		})

		t.Run("METRICS_USERNAMEが空文字の場合、notEmptyでエラーになること", func(t *testing.T) {
			setEnvVarsForTesting(t)
			t.Setenv("METRICS_USERNAME", "") // 空文字は required を通過するが notEmpty で弾く

			actual, err := New()
			assert.Nil(t, actual)
			require.ErrorContains(t, err, "METRICS_USERNAME")
		})

		t.Run("METRICS_PASSWORDが空文字の場合、notEmptyでエラーになること", func(t *testing.T) {
			setEnvVarsForTesting(t)
			t.Setenv("METRICS_PASSWORD", "") // 空文字は required を通過するが notEmpty で弾く

			actual, err := New()
			assert.Nil(t, actual)
			require.ErrorContains(t, err, "METRICS_PASSWORD")
		})
	})
}

func Test_validateConfig(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		cfg := mockLoader(t)

		err := validateConfig(cfg)
		require.NoError(t, err)
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("アプリケーション設定でエラーが発生する場合、エラーが返されること", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.App.Mode = "invalid_mode" // 無効なアプリケーションモード

			err := validateConfig(cfg)
			require.ErrorIs(t, err, ErrInvalidAppMode)
		})

		t.Run("サーバー設定でエラーが発生する場合、エラーが返されること", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Server.ReadHeaderTimeout = 0 // 無効なReadHeaderTimeout

			err := validateConfig(cfg)
			require.ErrorIs(t, err, ErrInvalidReadHeaderTimeout)
		})

		t.Run("DB設定でエラーが発生する場合、エラーが返されること", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Database.SlowQueryWarnThreshold = -1 // 無効なスロークエリ警告閾値

			err := validateConfig(cfg)
			require.ErrorIs(t, err, ErrInvalidSlowQueryWarnThreshold)
		})

		t.Run("DBコネクション設定でエラーが発生する場合、エラーが返されること", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.DBConnection.MinConns = cfg.DBConnection.MaxConns + 1 // MinConnsがMaxConnsを超える

			err := validateConfig(cfg)
			require.ErrorIs(t, err, ErrInvalidExceedMaxConns)
		})

		t.Run("セキュリティ設定でエラーが発生する場合、エラーが返されること", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Security.AllowedOrigins = []string{} // 空のAllowedOrigins

			err := validateConfig(cfg)
			require.ErrorIs(t, err, ErrEmptyAllowedOrigins)
		})

		t.Run("メトリクス設定でエラーが発生する場合、エラーが返されること", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Metrics.Port = MinPort - 1 // 無効なポート番号

			err := validateConfig(cfg)
			require.ErrorIs(t, err, ErrInvalidMetricsPortRange)
		})
	})
}

//nolint:paralleltest // embeddedAppEnv パッケージ変数を操作するため並列化不可
func Test_validateEmbeddedEnv(t *testing.T) {
	restore := func() func() {
		saved := embeddedAppEnv
		return func() { embeddedAppEnv = saved }
	}

	t.Run("正常系", func(t *testing.T) {
		t.Run("productionモードで本番素性(prd)の埋め込みenvの場合、エラーが返されないこと", func(t *testing.T) {
			defer restore()()
			embeddedAppEnv = EnvProduction

			cfg := mockLoader(t)
			cfg.App.Mode = ProductionMode

			require.NoError(t, validateConfig(cfg))
		})

		t.Run("productionモードで未知の環境素性の場合はdeny-listにないため許容されること", func(t *testing.T) {
			defer restore()()
			embeddedAppEnv = "unknown-future-env"

			cfg := mockLoader(t)
			cfg.App.Mode = ProductionMode

			require.NoError(t, validateConfig(cfg))
		})

		t.Run("developmentモードでは非本番素性の埋め込みenvでも許容されること", func(t *testing.T) {
			defer restore()()
			embeddedAppEnv = EnvLocal

			cfg := mockLoader(t)
			cfg.App.Mode = DevelopmentMode

			require.NoError(t, validateConfig(cfg))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("productionモードでlocal素性の埋め込みenvの場合、エラーが返されること", func(t *testing.T) {
			defer restore()()
			embeddedAppEnv = EnvLocal

			cfg := mockLoader(t)
			cfg.App.Mode = ProductionMode

			require.ErrorIs(t, validateConfig(cfg), ErrEmbeddedEnvMismatch)
		})

		t.Run("productionモードでci素性の埋め込みenvの場合、エラーが返されること", func(t *testing.T) {
			defer restore()()
			embeddedAppEnv = EnvCI

			cfg := mockLoader(t)
			cfg.App.Mode = ProductionMode

			require.ErrorIs(t, validateConfig(cfg), ErrEmbeddedEnvMismatch)
		})

		t.Run("productionモードでtest素性の埋め込みenvの場合、エラーが返されること", func(t *testing.T) {
			defer restore()()
			embeddedAppEnv = EnvTest

			cfg := mockLoader(t)
			cfg.App.Mode = ProductionMode

			require.ErrorIs(t, validateConfig(cfg), ErrEmbeddedEnvMismatch)
		})

		t.Run("productionモードでdevelopment素性の埋め込みenvの場合、エラーが返されること", func(t *testing.T) {
			defer restore()()
			embeddedAppEnv = EnvDevelopment

			cfg := mockLoader(t)
			cfg.App.Mode = ProductionMode

			require.ErrorIs(t, validateConfig(cfg), ErrEmbeddedEnvMismatch)
		})

		t.Run("productionモードで埋め込みenvが空の場合、エラーが返されること", func(t *testing.T) {
			defer restore()()
			embeddedAppEnv = ""

			cfg := mockLoader(t)
			cfg.App.Mode = ProductionMode

			require.ErrorIs(t, validateConfig(cfg), ErrEmbeddedEnvMismatch)
		})
	})
}

func Test_validateApplicationConfig(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		cfg := mockLoader(t)
		err := validateApplicationConfig(cfg.App)
		require.NoError(t, err)
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()
		t.Run("無効なアプリケーションモード", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.App.Mode = "invalid_mode" // 無効なアプリケーションモード

			err := validateApplicationConfig(cfg.App)
			require.ErrorIs(t, err, ErrInvalidAppMode)
		})

		t.Run("無効なログレベル", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.App.LogLevel = "invalid_level"

			err := validateApplicationConfig(cfg.App)
			require.ErrorIs(t, err, ErrInvalidLogLevel)
		})
	})
}

func Test_validateServerConfig(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		cfg := mockLoader(t)
		err := validateServerConfig(cfg.Server)
		require.NoError(t, err)
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()
		t.Run("無効なポート番号", func(t *testing.T) {
			t.Parallel()

			t.Run("ポート番号がMinPort未満の場合", func(t *testing.T) {
				t.Parallel()
				cfg := mockLoader(t)
				cfg.Server.Port = MinPort - 1 // 無効なポート番号（下限境界）

				err := validateServerConfig(cfg.Server)
				require.ErrorIs(t, err, ErrInvalidPortRange)
			})

			t.Run("ポート番号がMaxPortを超えている場合", func(t *testing.T) {
				t.Parallel()
				cfg := mockLoader(t)
				cfg.Server.Port = MaxPort + 1 // 無効なポート番号（上限境界）

				err := validateServerConfig(cfg.Server)
				require.ErrorIs(t, err, ErrInvalidPortRange)
			})
		})

		t.Run("ReadHeaderTimeoutが無効な場合", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Server.ReadHeaderTimeout = 0 // 無効なReadHeaderTimeout

			err := validateServerConfig(cfg.Server)
			require.ErrorIs(t, err, ErrInvalidReadHeaderTimeout)
		})

		t.Run("ReadTimeoutが無効な場合", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Server.ReadTimeout = 0 // 無効なReadTimeout

			err := validateServerConfig(cfg.Server)
			require.ErrorIs(t, err, ErrInvalidReadTimeout)
		})

		t.Run("WriteTimeoutが無効な場合", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Server.WriteTimeout = 0 // 無効なWriteTimeout

			err := validateServerConfig(cfg.Server)
			require.ErrorIs(t, err, ErrInvalidWriteTimeout)
		})

		t.Run("IdleTimeoutが無効な場合", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Server.IdleTimeout = 0 // 無効なIdleTimeout

			err := validateServerConfig(cfg.Server)
			require.ErrorIs(t, err, ErrInvalidIdleTimeout)
		})

		t.Run("ReadHeaderTimeoutがReadTimeoutを超えている場合", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Server.ReadHeaderTimeout = cfg.Server.ReadTimeout + cfg.Server.ReadTimeout

			err := validateServerConfig(cfg.Server)
			require.ErrorIs(t, err, ErrReadHeaderTimeoutExceedsReadTimeout)
		})

		t.Run("WriteTimeoutがRequestTimeout未満の場合", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Server.WriteTimeout = cfg.Server.RequestTimeout - time.Second // RequestTimeout より 1s 短い

			err := validateServerConfig(cfg.Server)
			require.ErrorIs(t, err, ErrWriteTimeoutBelowRequestTimeout)
		})

		t.Run("WriteTimeoutがRequestTimeoutと同値の場合は許容されること", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Server.WriteTimeout = cfg.Server.RequestTimeout // 境界値: 等値は有効

			err := validateServerConfig(cfg.Server)
			require.NoError(t, err)
		})
	})
}

func Test_validateMetricsConfig(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("デフォルトのポート番号は許容されること", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			err := validateMetricsConfig(cfg.Metrics)
			require.NoError(t, err)
		})

		t.Run("ポート番号がMinPortの場合は許容されること", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Metrics.Port = MinPort // 有効範囲の下限境界
			err := validateMetricsConfig(cfg.Metrics)
			require.NoError(t, err)
		})

		t.Run("ポート番号がMaxPortの場合は許容されること", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Metrics.Port = MaxPort // 有効範囲の上限境界
			err := validateMetricsConfig(cfg.Metrics)
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ポート番号がMinPort未満の場合", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Metrics.Port = MinPort - 1 // 無効なポート番号（下限境界）
			err := validateMetricsConfig(cfg.Metrics)
			require.ErrorIs(t, err, ErrInvalidMetricsPortRange)
		})

		t.Run("ポート番号がMaxPortを超えている場合", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Metrics.Port = MaxPort + 1 // 無効なポート番号（上限境界）
			err := validateMetricsConfig(cfg.Metrics)
			require.ErrorIs(t, err, ErrInvalidMetricsPortRange)
		})
	})
}

func Test_validateServerShutdown(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("shutdownがrequestを上回る場合は成功する", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateServerShutdown(90*time.Second, 60*time.Second))
		})

		t.Run("shutdownがrequestと同値の場合は許容される", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateServerShutdown(60*time.Second, 60*time.Second)) // 境界値: 等値は有効
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("shutdownがrequest未満の場合はErrShutdownTimeoutBelowRequestTimeoutを返す", func(t *testing.T) {
			t.Parallel()
			err := validateServerShutdown(60*time.Second-time.Second, 60*time.Second)
			require.ErrorIs(t, err, ErrShutdownTimeoutBelowRequestTimeout)
		})
	})
}

func Test_validateUploadBodyLimit(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ボディ上限がアップロード上限を上回る場合は成功する", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateUploadBodyLimit(6, 5_242_880))
		})

		t.Run("ボディ上限がアップロード上限を1バイト上回る場合は成功する", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateUploadBodyLimit(1, BytesPerMB-1)) // 境界値: 上回れば可
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ボディ上限がアップロード上限と同値の場合はErrBodyLimitBelowMaxUploadBytesを返す", func(t *testing.T) {
			t.Parallel()
			err := validateUploadBodyLimit(1, BytesPerMB) // 境界値: 等値はマルチパート分の余裕が無く不可
			require.ErrorIs(t, err, ErrBodyLimitBelowMaxUploadBytes)
		})

		t.Run("ボディ上限がアップロード上限を下回る場合はErrBodyLimitBelowMaxUploadBytesを返す", func(t *testing.T) {
			t.Parallel()
			err := validateUploadBodyLimit(5, 5_242_880)
			require.ErrorIs(t, err, ErrBodyLimitBelowMaxUploadBytes)
		})
	})
}

func Test_validateDatabaseConfig(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		cfg := mockLoader(t)
		err := validateDatabaseConfig(cfg.Database)
		require.NoError(t, err)
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()
		t.Run("無効なデータベースポート番号", func(t *testing.T) {
			t.Parallel()

			t.Run("ポート番号がMinPort未満の場合", func(t *testing.T) {
				t.Parallel()
				cfg := mockLoader(t)
				cfg.Database.Port = MinPort - 1 // 無効なデータベースポート番号

				err := validateDatabaseConfig(cfg.Database)
				require.ErrorIs(t, err, ErrInvalidDBPortRange)
			})

			t.Run("ポート番号がMaxPortを超えている場合", func(t *testing.T) {
				t.Parallel()
				cfg := mockLoader(t)
				cfg.Database.Port = MaxPort + 1 // 無効なデータベースポート番号

				err := validateDatabaseConfig(cfg.Database)
				require.ErrorIs(t, err, ErrInvalidDBPortRange)
			})
		})

		t.Run("無効なデータベースPingタイムアウト", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Database.PingTimeout = 0 // 無効なデータベースPingタイムアウト

			err := validateDatabaseConfig(cfg.Database)
			require.ErrorIs(t, err, ErrInvalidDBPingTimeout)
		})

		t.Run("無効なスロークエリ警告閾値", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Database.SlowQueryWarnThreshold = -1 // 無効なスロークエリ警告閾値

			err := validateDatabaseConfig(cfg.Database)
			require.ErrorIs(t, err, ErrInvalidSlowQueryWarnThreshold)
		})
	})
}

func Test_validateDBConnectionConfig(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		cfg := mockLoader(t)
		err := validateDBConnectionConfig(cfg.DBConnection)
		require.NoError(t, err)
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()
		t.Run("MinConnsがMaxConnsを超えている場合、エラーが返されること", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.DBConnection.MinConns = cfg.DBConnection.MaxConns + 1 // MinConnsがMaxConnsを超える

			err := validateDBConnectionConfig(cfg.DBConnection)
			require.ErrorIs(t, err, ErrInvalidExceedMaxConns)
		})
	})
}

func Test_validateSecurityConfig(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既定の許可オリジンの場合、エラーが返されないこと", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			err := validateSecurityConfig(cfg.Security)
			require.NoError(t, err)
		})

		t.Run("127.0.0.1へのHTTPは許可されること", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Security.AllowedOrigins = []string{"http://127.0.0.1"} // localhost 同等のループバック

			err := validateSecurityConfig(cfg.Security)
			require.NoError(t, err)
		})

		t.Run("localhost以外でもHTTPSは許可されること", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Security.AllowedOrigins = []string{"https://example.com"} // 非ループバックでも HTTPS なら許可

			err := validateSecurityConfig(cfg.Security)
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()
		t.Run("AllowedOriginsが空の場合", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Security.AllowedOrigins = []string{} // 空のAllowedOrigins

			err := validateSecurityConfig(cfg.Security)
			require.ErrorIs(t, err, ErrEmptyAllowedOrigins)
		})

		t.Run("localhost以外でHTTPが許可されている場合", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			cfg.Security.AllowedOrigins = []string{"http://example.com"} // localhost以外のHTTP

			err := validateSecurityConfig(cfg.Security)
			require.ErrorIs(t, err, ErrHTTPOnlyAllowedForLocalhost)
		})

		t.Run("オリジンのURLパースに失敗した場合", func(t *testing.T) {
			t.Parallel()
			cfg := mockLoader(t)
			// 制御文字を含む URL は url.Parse がエラーを返す。
			cfg.Security.AllowedOrigins = []string{"http://example.com/\x7f"}

			err := validateSecurityConfig(cfg.Security)
			require.ErrorIs(t, err, ErrHTTPOnlyAllowedForLocalhost)
		})
	})
}

func Test_buildStatusCodeSet(t *testing.T) {
	t.Parallel()

	codes := []int{200, 400, 500}
	expected := map[int]bool{
		200: true,
		400: true,
		500: true,
	}

	actual := buildStatusCodeSet(codes)
	assert.Equal(t, expected, actual)
}

func Test_parseCIDR(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な CIDR の場合、解析済みの *net.IPNet を返す", func(t *testing.T) {
			t.Parallel()

			actual, err := parseCIDR("192.168.0.0/16")
			require.NoError(t, err)
			require.NotNil(t, actual)
			assert.Equal(t, "192.168.0.0/16", actual.String())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不正な CIDR の場合、ErrFailedToParseCIDR を返す", func(t *testing.T) {
			t.Parallel()

			actual, err := parseCIDR("invalid_cidr")
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrFailedToParseCIDR)
		})
	})
}

func Test_validatePortRange(t *testing.T) {
	t.Parallel()

	sentinel := ErrInvalidPortRange

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("最小ポートの場合、nil を返す", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, validatePortRange(MinPort, sentinel))
		})

		t.Run("最大ポートの場合、nil を返す", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, validatePortRange(MaxPort, sentinel))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("最小ポート未満の場合、渡したエラーを返す", func(t *testing.T) {
			t.Parallel()

			require.ErrorIs(t, validatePortRange(MinPort-1, sentinel), sentinel)
		})

		t.Run("最大ポート超過の場合、渡したエラーを返す", func(t *testing.T) {
			t.Parallel()

			require.ErrorIs(t, validatePortRange(MaxPort+1, sentinel), sentinel)
		})
	})
}

func TestValidateServerShutdown(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("シャットダウン猶予がリクエスト予算を上回る場合、nilを返す", func(t *testing.T) {
			t.Parallel()

			appCfg := &ApplicationConfig{shutdownTimeout: 90 * time.Second}
			srvCfg := &ServerConfig{requestTimeout: 60 * time.Second}

			require.NoError(t, ValidateServerShutdown(appCfg, srvCfg))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("シャットダウン猶予がリクエスト予算を下回る場合、ErrShutdownTimeoutBelowRequestTimeoutを返す", func(t *testing.T) {
			t.Parallel()

			appCfg := &ApplicationConfig{shutdownTimeout: 60*time.Second - time.Second}
			srvCfg := &ServerConfig{requestTimeout: 60 * time.Second}

			err := ValidateServerShutdown(appCfg, srvCfg)
			require.ErrorIs(t, err, ErrShutdownTimeoutBelowRequestTimeout)
		})
	})
}

func TestValidateUploadBodyLimit(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ボディ上限がアップロード上限を上回る場合、nilを返す", func(t *testing.T) {
			t.Parallel()

			srvCfg := &ServerConfig{bodyLimitMB: 6}
			objCfg := &ObjectStorageConfig{maxUploadBytes: 5 * BytesPerMB}

			require.NoError(t, ValidateUploadBodyLimit(srvCfg, objCfg))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ボディ上限がアップロード上限と同値の場合、ErrBodyLimitBelowMaxUploadBytesを返す", func(t *testing.T) {
			t.Parallel()

			srvCfg := &ServerConfig{bodyLimitMB: 5}
			objCfg := &ObjectStorageConfig{maxUploadBytes: 5 * BytesPerMB}

			err := ValidateUploadBodyLimit(srvCfg, objCfg)
			require.ErrorIs(t, err, ErrBodyLimitBelowMaxUploadBytes)
		})
	})
}

func Test_isActiveExporter(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("exporter名が指定されている場合、trueを返す", func(t *testing.T) {
			t.Parallel()

			assert.True(t, isActiveExporter("otlp"))
		})

		t.Run("未設定を表す空文字の場合、falseを返す", func(t *testing.T) {
			t.Parallel()

			assert.False(t, isActiveExporter(""))
		})

		t.Run("明示的な無効化を表すnoneの場合、falseを返す", func(t *testing.T) {
			t.Parallel()

			assert.False(t, isActiveExporter(exporterNone))
		})

		t.Run("noneと大文字小文字だけが異なる場合、falseを返す", func(t *testing.T) {
			t.Parallel()

			assert.False(t, isActiveExporter("NoNe"))
		})
	})
}
