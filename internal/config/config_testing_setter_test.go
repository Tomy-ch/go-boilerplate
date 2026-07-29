package config

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplicationConfig_SetApplicationMode(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定したモードへ差し替わり、クリーンアップで元のモードへ戻る", func(t *testing.T) {
			t.Parallel()
			app := MockConfigForTest(t).app

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				app.SetApplicationMode(t, ProductionMode)
				assert.Equal(t, ProductionMode, app.Mode())
			})

			assert.Equal(t, expectedApplicationMode, app.Mode())
		})
	})
}

func TestApplicationConfig_SetApplicationEnv(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定した環境へ差し替わり、クリーンアップで元の環境へ戻る", func(t *testing.T) {
			t.Parallel()
			app := MockConfigForTest(t).app

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				app.SetApplicationEnv(t, EnvStaging)
				assert.Equal(t, EnvStaging, app.Env())
			})

			assert.Equal(t, expectedApplicationEnv, app.Env())
		})
	})
}

func TestApplicationConfig_SetApplicationLogLevel(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定したログレベルへ差し替わり、クリーンアップで元のログレベルへ戻る", func(t *testing.T) {
			t.Parallel()
			app := MockConfigForTest(t).app

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				app.SetApplicationLogLevel(t, LogLevelWarn)
				assert.Equal(t, LogLevelWarn, app.LogLevel())
			})

			assert.Equal(t, expectedApplicationLogLevel, app.LogLevel())
		})
	})
}

func TestServerConfig_SetServerPort(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定したポートへ差し替わり、クリーンアップで元のポートへ戻る", func(t *testing.T) {
			t.Parallel()
			server := MockConfigForTest(t).server

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				server.SetServerPort(t, 18080)
				assert.Equal(t, 18080, server.Port())
			})

			assert.Equal(t, expectedServerPort, server.Port())
		})
	})
}

func TestMetricsConfig_SetMetricsPort(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定したポートへ差し替わり、クリーンアップで元のポートへ戻る", func(t *testing.T) {
			t.Parallel()
			metrics := MockConfigForTest(t).metrics

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				metrics.SetMetricsPort(t, 16060)
				assert.Equal(t, 16060, metrics.Port())
			})

			assert.Equal(t, expectedMetricsPort, metrics.Port())
		})
	})
}

func TestObservabilityConfig_SetObservabilityMaskedDBQueryArgs(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定したマスク設定へ差し替わり、クリーンアップで元の設定へ戻る", func(t *testing.T) {
			t.Parallel()
			observability := MockConfigForTest(t).observability

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				observability.SetObservabilityMaskedDBQueryArgs(t, !expectedObservabilityMaskedDBQueryArgs)
				assert.Equal(t, !expectedObservabilityMaskedDBQueryArgs, observability.MaskedDBQueryArgs())
			})

			assert.Equal(t, expectedObservabilityMaskedDBQueryArgs, observability.MaskedDBQueryArgs())
		})
	})
}

func TestObservabilityConfig_SetObservabilityTracesExporter(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("無効値へ差し替えると trace 送出が無効になり、クリーンアップで有効へ戻る", func(t *testing.T) {
			t.Parallel()
			observability := MockConfigForTest(t).observability

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				observability.SetObservabilityTracesExporter(t, exporterNone)
				assert.False(t, observability.TracesEnabled())
			})

			assert.True(t, observability.TracesEnabled())
		})
	})
}

func TestObservabilityConfig_SetObservabilityMetricsExporter(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("無効値へ差し替えると metric 送出が無効になり、クリーンアップで有効へ戻る", func(t *testing.T) {
			t.Parallel()
			observability := MockConfigForTest(t).observability

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				observability.SetObservabilityMetricsExporter(t, exporterNone)
				assert.False(t, observability.MetricsEnabled())
			})

			assert.True(t, observability.MetricsEnabled())
		})
	})
}

func TestObservabilityConfig_SetObservabilityLogsExporter(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("無効値へ差し替えると log 送出が無効になり、クリーンアップで有効へ戻る", func(t *testing.T) {
			t.Parallel()
			observability := MockConfigForTest(t).observability

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				observability.SetObservabilityLogsExporter(t, exporterNone)
				assert.False(t, observability.LogsEnabled())
			})

			assert.True(t, observability.LogsEnabled())
		})
	})
}

func TestObservabilityConfig_SetObservabilityOTLPProtocol(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定したプロトコルへ差し替わり、クリーンアップで元のプロトコルへ戻る", func(t *testing.T) {
			t.Parallel()
			observability := MockConfigForTest(t).observability

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				observability.SetObservabilityOTLPProtocol(t, "grpc")
				assert.Equal(t, "grpc", observability.OTLPProtocol())
			})

			assert.Equal(t, expectedObservabilityOTLPProtocol, observability.OTLPProtocol())
		})
	})
}

func TestObservabilityConfig_SetObservabilityOTLPEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定したエンドポイントへ差し替わり、クリーンアップで元のエンドポイントへ戻る", func(t *testing.T) {
			t.Parallel()
			observability := MockConfigForTest(t).observability

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				observability.SetObservabilityOTLPEndpoint(t, "http://collector.test:4318")
				assert.Equal(t, "http://collector.test:4318", observability.OTLPEndpoint())
			})

			assert.Equal(t, expectedObservabilityOTLPEndpoint, observability.OTLPEndpoint())
		})
	})
}

func TestDatabaseConfig_SetDatabaseHost(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定したホストへ差し替わり、クリーンアップで元のホストへ戻る", func(t *testing.T) {
			t.Parallel()
			database := MockConfigForTest(t).database

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				database.SetDatabaseHost(t, "db.test")
				assert.Equal(t, "db.test", database.Host())
			})

			assert.Equal(t, expectedDBHost, database.Host())
		})
	})
}

func TestDatabaseConfig_SetDatabaseName(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定したデータベース名へ差し替わり、クリーンアップで元の名前へ戻る", func(t *testing.T) {
			t.Parallel()
			database := MockConfigForTest(t).database

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				database.SetDatabaseName(t, "other-db")
				assert.Equal(t, "other-db", database.DBName())
			})

			assert.Equal(t, expectedDBName, database.DBName())
		})
	})
}

func TestDBConnectionConfig_SetMaxConns(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定した最大コネクション数へ差し替わり、クリーンアップで元の値へ戻る", func(t *testing.T) {
			t.Parallel()
			dbconnection := MockConfigForTest(t).dbconnection

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				dbconnection.SetMaxConns(t, int32(20))
				assert.Equal(t, int32(20), dbconnection.MaxConns())
			})

			assert.Equal(t, expectedDBMaxConnsInt32, dbconnection.MaxConns())
		})
	})
}

func TestSecurityConfig_SetCIDR(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定したCIDRへ差し替わり、クリーンアップで元のCIDRへ戻る", func(t *testing.T) {
			t.Parallel()
			security := MockConfigForTest(t).security
			_, cidr, err := net.ParseCIDR("10.0.0.0/8")
			require.NoError(t, err)

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				security.SetCIDR(t, cidr)
				assert.Equal(t, cidr, security.CIDR())
			})

			assert.Equal(t, expectedCIDR, security.CIDR())
		})
	})
}

func TestSecureCookieConfig_SetSameSite(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定したSameSiteへ差し替わり、クリーンアップで元の値へ戻る", func(t *testing.T) {
			t.Parallel()
			secureCookie := MockConfigForTest(t).secureCookie

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				secureCookie.SetSameSite(t, "Lax")
				assert.Equal(t, "Lax", secureCookie.SameSite())
			})

			assert.Equal(t, expectedSecureCookieSameSite, secureCookie.SameSite())
		})
	})
}

func TestSecureCookieConfig_SetDomain(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定したDomainへ差し替わり、クリーンアップで元の値へ戻る", func(t *testing.T) {
			t.Parallel()
			secureCookie := MockConfigForTest(t).secureCookie

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				secureCookie.SetDomain(t, "example.test")
				assert.Equal(t, "example.test", secureCookie.Domain())
			})

			assert.Equal(t, expectedSecureCookieDomain, secureCookie.Domain())
		})
	})
}

func TestWorkerConfig_SetHealthListenAddr(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定した待ち受けアドレスへ差し替わり、クリーンアップで元のアドレスへ戻る", func(t *testing.T) {
			t.Parallel()
			worker := MockConfigForTest(t).worker

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				worker.SetHealthListenAddr(t, "127.0.0.1:0")
				assert.Equal(t, "127.0.0.1:0", worker.HealthListenAddr())
			})

			assert.Equal(t, expectedWorkerHealthListenAddr, worker.HealthListenAddr())
		})
	})
}

func TestOutboxConfig_SetOutboxBatchSize(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定したバッチサイズへ差し替わり、クリーンアップで元の値へ戻る", func(t *testing.T) {
			t.Parallel()
			outbox := MockConfigForTest(t).outbox

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				outbox.SetOutboxBatchSize(t, 7)
				assert.Equal(t, 7, outbox.BatchSize())
			})

			assert.Equal(t, expectedOutboxBatchSize, outbox.BatchSize())
		})
	})
}

func TestOutboxConfig_SetOutboxPollInterval(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定したポーリング間隔へ差し替わり、クリーンアップで元の値へ戻る", func(t *testing.T) {
			t.Parallel()
			outbox := MockConfigForTest(t).outbox

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				outbox.SetOutboxPollInterval(t, 3*time.Second)
				assert.Equal(t, 3*time.Second, outbox.PollInterval())
			})

			assert.Equal(t, expectedOutboxPollInterval, outbox.PollInterval())
		})
	})
}

func TestOutboxConfig_SetOutboxErrorBackoff(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定したエラー時待機時間へ差し替わり、クリーンアップで元の値へ戻る", func(t *testing.T) {
			t.Parallel()
			outbox := MockConfigForTest(t).outbox

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				outbox.SetOutboxErrorBackoff(t, 9*time.Second)
				assert.Equal(t, 9*time.Second, outbox.ErrorBackoff())
			})

			assert.Equal(t, expectedOutboxErrorBackoff, outbox.ErrorBackoff())
		})
	})
}

func TestOutboxConfig_SetOutboxEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定したエンドポイントへ差し替わり、クリーンアップで元の値へ戻る", func(t *testing.T) {
			t.Parallel()
			outbox := MockConfigForTest(t).outbox

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				outbox.SetOutboxEndpoint(t, "http://relay.test:8080")
				assert.Equal(t, "http://relay.test:8080", outbox.Endpoint())
			})

			assert.Equal(t, expectedOutboxEndpoint, outbox.Endpoint())
		})
	})
}

func TestAuthConfig_SetAuthIssuer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定したissuerへ差し替わり、クリーンアップで元の値へ戻る", func(t *testing.T) {
			t.Parallel()
			auth := MockConfigForTest(t).auth

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				auth.SetAuthIssuer(t, "https://issuer.test")
				assert.Equal(t, "https://issuer.test", auth.Issuer())
			})

			assert.Equal(t, expectedAuthIssuer, auth.Issuer())
		})
	})
}

func TestAuthConfig_SetAuthAudience(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定したaudienceへ差し替わり、クリーンアップで元の値へ戻る", func(t *testing.T) {
			t.Parallel()
			auth := MockConfigForTest(t).auth

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				auth.SetAuthAudience(t, "test-audience")
				assert.Equal(t, "test-audience", auth.Audience())
			})

			assert.Equal(t, expectedAuthAudience, auth.Audience())
		})
	})
}

func TestAuthConfig_SetAuthJWKSURL(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定したJWKS URLへ差し替わり、クリーンアップで元の値へ戻る", func(t *testing.T) {
			t.Parallel()
			auth := MockConfigForTest(t).auth

			t.Run("一時的に差し替える", func(t *testing.T) { //nolint:paralleltest // Cleanup 発火後の復元を親で検証するため同期実行する
				auth.SetAuthJWKSURL(t, "https://issuer.test/jwks")
				assert.Equal(t, "https://issuer.test/jwks", auth.JWKSURL())
			})

			assert.Equal(t, expectedAuthJWKSURL, auth.JWKSURL())
		})
	})
}
