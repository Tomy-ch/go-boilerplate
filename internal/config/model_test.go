package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstructor(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		cfg := MockConfigForTest(t)

		t.Run("OS設定のコンストラクタが内部フィールドへの参照を返す", func(t *testing.T) {
			t.Parallel()
			osCfg := NewOperatingSystemConfig(cfg)
			assert.Same(t, &cfg.os, osCfg)
		})

		t.Run("サーバー設定のコンストラクタが内部フィールドへの参照を返す", func(t *testing.T) {
			t.Parallel()
			serverCfg := NewServerConfig(cfg)
			assert.Same(t, &cfg.server, serverCfg)
		})

		t.Run("メトリクス設定のコンストラクタが内部フィールドへの参照を返す", func(t *testing.T) {
			t.Parallel()
			metricsCfg := NewMetricsConfig(cfg)
			assert.Same(t, &cfg.metrics, metricsCfg)
		})

		t.Run("オブザーバビリティ設定のコンストラクタが内部フィールドへの参照を返す", func(t *testing.T) {
			t.Parallel()
			observabilityCfg := NewObservabilityConfig(cfg)
			assert.Same(t, &cfg.observability, observabilityCfg)
		})

		t.Run("アプリケーション設定のコンストラクタが内部フィールドへの参照を返す", func(t *testing.T) {
			t.Parallel()
			appCfg := NewApplicationConfig(cfg)
			assert.Same(t, &cfg.app, appCfg)
		})

		t.Run("データベース設定のコンストラクタが内部フィールドへの参照を返す", func(t *testing.T) {
			t.Parallel()
			dbCfg := NewDatabaseConfig(cfg)
			assert.Same(t, &cfg.database, dbCfg)
		})

		t.Run("DBコネクション設定のコンストラクタが内部フィールドへの参照を返す", func(t *testing.T) {
			t.Parallel()
			dbConnCfg := NewDBConnectionConfig(cfg)
			assert.Same(t, &cfg.dbconnection, dbConnCfg)
		})

		t.Run("セキュリティ設定のコンストラクタが内部フィールドへの参照を返す", func(t *testing.T) {
			t.Parallel()
			securityCfg := NewSecurityConfig(cfg)
			assert.Same(t, &cfg.security, securityCfg)
		})

		t.Run("セキュアCookie設定のコンストラクタが内部フィールドへの参照を返す", func(t *testing.T) {
			t.Parallel()
			secureCookieCfg := NewSecureCookieConfig(cfg)
			assert.Same(t, &cfg.secureCookie, secureCookieCfg)
		})

		t.Run("worker設定のコンストラクタが内部フィールドへの参照を返す", func(t *testing.T) {
			t.Parallel()
			workerCfg := NewWorkerConfig(cfg)
			assert.Same(t, &cfg.worker, workerCfg)
		})

		t.Run("outbox設定のコンストラクタが内部フィールドへの参照を返す", func(t *testing.T) {
			t.Parallel()
			outboxCfg := NewOutboxConfig(cfg)
			assert.Same(t, &cfg.outbox, outboxCfg)
		})
	})
}

func TestGetterMethods(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		cfg := MockConfigForTest(t)

		t.Run("OS設定", func(t *testing.T) {
			t.Parallel()
			os := cfg.os
			t.Run("タイムゾーンを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedOSTimeZone, os.TimeZone())
			})
		})

		t.Run("サーバー設定", func(t *testing.T) {
			t.Parallel()
			server := cfg.server
			t.Run("ホストを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedServerHost, server.Host())
			})

			t.Run("ポートを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedServerPort, server.Port())
			})

			t.Run("ヘッダー読み取りタイムアウトを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedServerReadHeaderTimeout, server.ReadHeaderTimeout())
			})

			t.Run("読み取りタイムアウトを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedServerReadTimeout, server.ReadTimeout())
			})

			t.Run("書き込みタイムアウトを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedServerWriteTimeout, server.WriteTimeout())
			})

			t.Run("アイドルタイムアウトを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedServerIdleTimeout, server.IdleTimeout())
			})

			t.Run("ボディ上限MBを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedServerBodyLimitMB, server.BodyLimitMB())
			})

			t.Run("リクエストタイムアウトを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedServerRequestTimeout, server.RequestTimeout())
			})
		})

		t.Run("メトリクス設定", func(t *testing.T) {
			t.Parallel()
			metrics := cfg.metrics

			t.Run("ホストを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedMetricsHost, metrics.Host())
			})

			t.Run("ポートを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedMetricsPort, metrics.Port())
			})

			t.Run("ユーザー名を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedMetricsUserName, metrics.UserName())
			})

			t.Run("パスワードを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedMetricsPassword, metrics.Password())
			})
		})

		t.Run("オブザーバビリティ設定", func(t *testing.T) {
			t.Parallel()
			observability := cfg.observability

			t.Run("trace exporter が有効値なら有効を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.True(t, observability.TracesEnabled())
			})

			t.Run("metric exporter が有効値なら有効を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.True(t, observability.MetricsEnabled())
			})

			t.Run("log exporter が有効値なら有効を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.True(t, observability.LogsEnabled())
			})

			t.Run("いずれかの exporter が有効値なら全体有効を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.True(t, observability.Enabled())
			})

			t.Run("OTLPエンドポイントを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedObservabilityOTLPEndpoint, observability.OTLPEndpoint())
			})

			t.Run("OTLPプロトコルを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedObservabilityOTLPProtocol, observability.OTLPProtocol())
			})

			t.Run("DBクエリ引数マスク設定を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedObservabilityMaskedDBQueryArgs, observability.MaskedDBQueryArgs())
			})

			t.Run("対象ステータスコード集合を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedObservabilityTargetStatusCodeSet, observability.TargetStatusCodeSet())
			})
		})

		t.Run("アプリケーション設定", func(t *testing.T) {
			t.Parallel()
			app := cfg.app
			t.Run("環境を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedApplicationEnv, app.Env())
			})

			t.Run("名前を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedApplicationName, app.Name())
			})

			t.Run("モードを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedApplicationMode, app.Mode())
			})

			t.Run("ログレベルを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedApplicationLogLevel, app.LogLevel())
			})

			t.Run("シャットダウンタイムアウトを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedAppShutdownTimeout, app.ShutdownTimeout())
			})
		})

		t.Run("データベース設定", func(t *testing.T) { //nolint:dupl // 逐次t.Runが規約(table-driven禁止)。getter群が他設定ブロックと同型でも重複を許容
			t.Parallel()
			database := cfg.database
			t.Run("ドライバーを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBDriver, database.Driver())
			})

			t.Run("ホストを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBHost, database.Host())
			})

			t.Run("ポートを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBPort, database.Port())
			})

			t.Run("ユーザーを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBUser, database.User())
			})

			t.Run("パスワードを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBPassword, database.Password())
			})

			t.Run("データベース名を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBName, database.DBName())
			})

			t.Run("SSLモードを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBSSLMode, database.SSLMode())
			})

			t.Run("Pingタイムアウトを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBPingTimeout, database.PingTimeout())
			})

			t.Run("スロークエリ警告閾値を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBSlowQueryWarnThreshold, database.SlowQueryWarnThreshold())
			})

			t.Run("statement_timeout を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBStatementTimeout, database.StatementTimeout())
			})

			t.Run("lock_timeout を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBLockTimeout, database.LockTimeout())
			})

			t.Run("トランザクションリトライ最大試行回数を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBTxMaxRetries, database.TxMaxRetries())
			})

			t.Run("トランザクションリトライbackoff初期値を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBTxRetryBaseBackoff, database.TxRetryBaseBackoff())
			})

			t.Run("トランザクションリトライbackoff上限値を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBTxRetryMaxBackoff, database.TxRetryMaxBackoff())
			})
		})

		t.Run("DBコネクション設定", func(t *testing.T) {
			t.Parallel()
			connection := cfg.dbconnection
			t.Run("最大コネクション数を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBMaxConnsInt32, connection.MaxConns())
			})

			t.Run("最小コネクション数を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBMinConnsInt32, connection.MinConns())
			})

			t.Run("最大ライフタイムを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBMaxLifetime, connection.MaxLifetime())
			})

			t.Run("最大アイドル時間を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedDBMaxIdleTime, connection.MaxIdleTime())
			})
		})

		t.Run("セキュリティ設定", func(t *testing.T) {
			t.Parallel()
			security := cfg.security
			t.Run("許可オリジンを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(
					t,
					strings.Split(expectedAllowedOrigins, ","),
					security.AllowedOrigins(),
				)
			})

			t.Run("CIDRを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedCIDR, security.CIDR())
			})

			t.Run("ContentTypeNosniffを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedContentTypeNosniff, security.ContentTypeNosniff())
			})

			t.Run("XFrameOptionsを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedXFrameOptions, security.XFrameOptions())
			})

			t.Run("HSTS最大期間を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedHSTSMaxAge, security.HSTSMaxAge())
			})

			t.Run("HSTSサブドメイン除外設定を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedHSTSExcludeSubdomains, security.HSTSExcludeSubdomains())
			})

			t.Run("HSTSプリロード有効設定を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedHSTSPreloadEnabled, security.HSTSPreloadEnabled())
			})

			t.Run("リファラーポリシーを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedReferrerPolicy, security.ReferrerPolicy())
			})

			t.Run("bcryptコストを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedBcryptCost, security.BcryptCost())
			})
		})

		t.Run("セキュアCookie設定", func(t *testing.T) {
			t.Parallel()
			secureCookie := cfg.secureCookie

			t.Run("Secure属性を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedSecureCookieSecure, secureCookie.Secure())
			})

			t.Run("SameSite属性を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedSecureCookieSameSite, secureCookie.SameSite())
			})

			t.Run("ドメインを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedSecureCookieDomain, secureCookie.Domain())
			})
		})

		t.Run("worker設定", func(t *testing.T) { //nolint:dupl // 逐次t.Runが規約(table-driven禁止)。getter群が他設定ブロックと同型でも重複を許容
			t.Parallel()
			worker := cfg.worker

			t.Run("並行実行数を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedWorkerConcurrency, worker.Concurrency())
			})

			t.Run("最大インフライト数を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedWorkerMaxInFlight, worker.MaxInFlight())
			})

			t.Run("バッチサイズを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedWorkerBatchSize, worker.BatchSize())
			})

			t.Run("Extend周期を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedWorkerExtendInterval, worker.ExtendInterval())
			})

			t.Run("ドレインタイムアウトを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedWorkerDrainTimeout, worker.DrainTimeout())
			})

			t.Run("再配送回数の警告閾値を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedWorkerReceiveCountWarnThreshold, worker.ReceiveCountWarnThreshold())
			})

			t.Run("サーキット失敗閾値を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedWorkerCircuitFailureThreshold, worker.CircuitFailureThreshold())
			})

			t.Run("サーキットOpen初回backoffを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedWorkerCircuitOpenBackoffInitial, worker.CircuitOpenBackoffInitial())
			})

			t.Run("サーキットOpen backoff上限を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedWorkerCircuitOpenBackoffMax, worker.CircuitOpenBackoffMax())
			})

			t.Run("Half-open試行数を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedWorkerCircuitHalfOpenProbe, worker.CircuitHalfOpenProbe())
			})

			t.Run("health listener待ち受けアドレスを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedWorkerHealthListenAddr, worker.HealthListenAddr())
			})

			t.Run("進捗停滞判定時間を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedWorkerProgressStaleAfter, worker.ProgressStaleAfter())
			})

			t.Run("Nack初回backoffを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedWorkerNackBackoffInitial, worker.NackBackoffInitial())
			})

			t.Run("Nack backoff上限を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedWorkerNackBackoffMax, worker.NackBackoffMax())
			})
		})

		t.Run("outbox設定", func(t *testing.T) {
			t.Parallel()
			outbox := cfg.outbox

			t.Run("エンドポイントを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedOutboxEndpoint, outbox.Endpoint())
			})

			t.Run("ポーリング間隔を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedOutboxPollInterval, outbox.PollInterval())
			})

			t.Run("エラーbackoffを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedOutboxErrorBackoff, outbox.ErrorBackoff())
			})

			t.Run("バッチサイズを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedOutboxBatchSize, outbox.BatchSize())
			})
		})
	})
}

func TestApplicationConfig_IsProductionMode(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("本番環境モードの場合", func(t *testing.T) {
			t.Parallel()
			cfg := Config{}
			cfg.app.mode = ProductionMode
			assert.True(t, cfg.app.IsProductionMode())
		})

		t.Run("開発環境モードの場合", func(t *testing.T) {
			t.Parallel()
			cfg := Config{}
			cfg.app.mode = DevelopmentMode
			assert.False(t, cfg.app.IsProductionMode())
		})
	})
}

func TestApplicationConfig_IsDevelopmentMode(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("開発環境モードの場合", func(t *testing.T) {
			t.Parallel()
			cfg := Config{}
			cfg.app.mode = DevelopmentMode
			assert.True(t, cfg.app.IsDevelopmentMode())
		})

		t.Run("本番環境モードの場合", func(t *testing.T) {
			t.Parallel()
			cfg := Config{}
			cfg.app.mode = ProductionMode
			assert.False(t, cfg.app.IsDevelopmentMode())
		})
	})
}

func TestSecurityConfig_CIDR(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("cidrがnilの場合_nilを返す", func(t *testing.T) {
			t.Parallel()
			s := &SecurityConfig{}
			assert.Nil(t, s.CIDR())
		})
	})
}

func TestSecureCookieConfig_Secure(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("secureがnilの場合_nilを返す", func(t *testing.T) {
			t.Parallel()
			s := &SecureCookieConfig{}
			assert.Nil(t, s.Secure())
		})
	})
}

func TestObservabilityConfig_Enabled(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全部otlpなら全て有効", func(t *testing.T) {
			t.Parallel()
			o := &ObservabilityConfig{tracesExporter: "otlp", metricsExporter: "otlp", logsExporter: "otlp"}
			assert.True(t, o.TracesEnabled())
			assert.True(t, o.MetricsEnabled())
			assert.True(t, o.LogsEnabled())
			assert.True(t, o.Enabled())
		})

		t.Run("traceのみ有効", func(t *testing.T) {
			t.Parallel()
			o := &ObservabilityConfig{tracesExporter: "otlp", metricsExporter: "", logsExporter: ""}
			assert.True(t, o.TracesEnabled())
			assert.False(t, o.MetricsEnabled())
			assert.False(t, o.LogsEnabled())
			assert.True(t, o.Enabled())
		})

		t.Run("metricのみ有効", func(t *testing.T) {
			t.Parallel()
			o := &ObservabilityConfig{tracesExporter: "", metricsExporter: "otlp", logsExporter: ""}
			assert.False(t, o.TracesEnabled())
			assert.True(t, o.MetricsEnabled())
			assert.False(t, o.LogsEnabled())
			assert.True(t, o.Enabled())
		})

		t.Run("logのみ有効", func(t *testing.T) {
			t.Parallel()
			o := &ObservabilityConfig{tracesExporter: "", metricsExporter: "", logsExporter: "otlp"}
			assert.False(t, o.TracesEnabled())
			assert.False(t, o.MetricsEnabled())
			assert.True(t, o.LogsEnabled())
			assert.True(t, o.Enabled())
		})

		t.Run("全部空なら無効", func(t *testing.T) {
			t.Parallel()
			o := &ObservabilityConfig{tracesExporter: "", metricsExporter: "", logsExporter: ""}
			assert.False(t, o.TracesEnabled())
			assert.False(t, o.MetricsEnabled())
			assert.False(t, o.LogsEnabled())
			assert.False(t, o.Enabled())
		})

		t.Run("noneは無効として扱う", func(t *testing.T) {
			t.Parallel()
			o := &ObservabilityConfig{tracesExporter: "none", metricsExporter: "none", logsExporter: "none"}
			assert.False(t, o.TracesEnabled())
			assert.False(t, o.MetricsEnabled())
			assert.False(t, o.LogsEnabled())
			assert.False(t, o.Enabled())
		})

		t.Run("大文字NONEも無効として扱う", func(t *testing.T) {
			t.Parallel()
			o := &ObservabilityConfig{tracesExporter: "NONE", metricsExporter: "None", logsExporter: "nOnE"}
			assert.False(t, o.TracesEnabled())
			assert.False(t, o.MetricsEnabled())
			assert.False(t, o.LogsEnabled())
			assert.False(t, o.Enabled())
		})
	})
}
