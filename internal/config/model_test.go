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

		t.Run("認証設定のコンストラクタが内部フィールドへの参照を返す", func(t *testing.T) {
			t.Parallel()
			authCfg := NewAuthConfig(cfg)
			assert.Same(t, &cfg.auth, authCfg)
		})
	})
}

func TestOperatingSystemConfig_TimeZone(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("タイムゾーンを取得できる", func(t *testing.T) {
			t.Parallel()
			os := MockConfigForTest(t).os
			assert.Equal(t, expectedOSTimeZone, os.TimeZone())
		})
	})
}

func TestServerConfig_Host(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("ホストを取得できる", func(t *testing.T) {
			t.Parallel()
			server := MockConfigForTest(t).server
			assert.Equal(t, expectedServerHost, server.Host())
		})
	})
}

func TestServerConfig_Port(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("ポートを取得できる", func(t *testing.T) {
			t.Parallel()
			server := MockConfigForTest(t).server
			assert.Equal(t, expectedServerPort, server.Port())
		})
	})
}

func TestServerConfig_ReadHeaderTimeout(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("ヘッダー読み取りタイムアウトを取得できる", func(t *testing.T) {
			t.Parallel()
			server := MockConfigForTest(t).server
			assert.Equal(t, expectedServerReadHeaderTimeout, server.ReadHeaderTimeout())
		})
	})
}

func TestServerConfig_ReadTimeout(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("読み取りタイムアウトを取得できる", func(t *testing.T) {
			t.Parallel()
			server := MockConfigForTest(t).server
			assert.Equal(t, expectedServerReadTimeout, server.ReadTimeout())
		})
	})
}

func TestServerConfig_WriteTimeout(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("書き込みタイムアウトを取得できる", func(t *testing.T) {
			t.Parallel()
			server := MockConfigForTest(t).server
			assert.Equal(t, expectedServerWriteTimeout, server.WriteTimeout())
		})
	})
}

func TestServerConfig_IdleTimeout(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("アイドルタイムアウトを取得できる", func(t *testing.T) {
			t.Parallel()
			server := MockConfigForTest(t).server
			assert.Equal(t, expectedServerIdleTimeout, server.IdleTimeout())
		})
	})
}

func TestServerConfig_BodyLimitMB(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("ボディ上限MBを取得できる", func(t *testing.T) {
			t.Parallel()
			server := MockConfigForTest(t).server
			assert.Equal(t, expectedServerBodyLimitMB, server.BodyLimitMB())
		})
	})
}

func TestServerConfig_RequestTimeout(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("リクエストタイムアウトを取得できる", func(t *testing.T) {
			t.Parallel()
			server := MockConfigForTest(t).server
			assert.Equal(t, expectedServerRequestTimeout, server.RequestTimeout())
		})
	})
}

func TestMetricsConfig_Host(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("ホストを取得できる", func(t *testing.T) {
			t.Parallel()
			metrics := MockConfigForTest(t).metrics
			assert.Equal(t, expectedMetricsHost, metrics.Host())
		})
	})
}

func TestMetricsConfig_Port(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("ポートを取得できる", func(t *testing.T) {
			t.Parallel()
			metrics := MockConfigForTest(t).metrics
			assert.Equal(t, expectedMetricsPort, metrics.Port())
		})
	})
}

func TestMetricsConfig_UserName(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("ユーザー名を取得できる", func(t *testing.T) {
			t.Parallel()
			metrics := MockConfigForTest(t).metrics
			assert.Equal(t, expectedMetricsUserName, metrics.UserName())
		})
	})
}

func TestMetricsConfig_Password(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("パスワードを取得できる", func(t *testing.T) {
			t.Parallel()
			metrics := MockConfigForTest(t).metrics
			assert.Equal(t, expectedMetricsPassword, metrics.Password())
		})
	})
}

func TestObservabilityConfig_TracesEnabled(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("trace exporter が有効値なら有効を取得できる", func(t *testing.T) {
			t.Parallel()
			observability := MockConfigForTest(t).observability
			assert.True(t, observability.TracesEnabled())
		})
	})
}

func TestObservabilityConfig_MetricsEnabled(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("metric exporter が有効値なら有効を取得できる", func(t *testing.T) {
			t.Parallel()
			observability := MockConfigForTest(t).observability
			assert.True(t, observability.MetricsEnabled())
		})
	})
}

func TestObservabilityConfig_LogsEnabled(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("log exporter が有効値なら有効を取得できる", func(t *testing.T) {
			t.Parallel()
			observability := MockConfigForTest(t).observability
			assert.True(t, observability.LogsEnabled())
		})
	})
}

func TestObservabilityConfig_OTLPEndpoint(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("OTLPエンドポイントを取得できる", func(t *testing.T) {
			t.Parallel()
			observability := MockConfigForTest(t).observability
			assert.Equal(t, expectedObservabilityOTLPEndpoint, observability.OTLPEndpoint())
		})
	})
}

func TestObservabilityConfig_OTLPProtocol(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("OTLPプロトコルを取得できる", func(t *testing.T) {
			t.Parallel()
			observability := MockConfigForTest(t).observability
			assert.Equal(t, expectedObservabilityOTLPProtocol, observability.OTLPProtocol())
		})
	})
}

func TestObservabilityConfig_MaskedDBQueryArgs(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("DBクエリ引数マスク設定を取得できる", func(t *testing.T) {
			t.Parallel()
			observability := MockConfigForTest(t).observability
			assert.Equal(t, expectedObservabilityMaskedDBQueryArgs, observability.MaskedDBQueryArgs())
		})
	})
}

func TestObservabilityConfig_TargetStatusCodeSet(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("対象ステータスコード集合を取得できる", func(t *testing.T) {
			t.Parallel()
			observability := MockConfigForTest(t).observability
			assert.Equal(t, expectedObservabilityTargetStatusCodeSet, observability.TargetStatusCodeSet())
		})
	})
}

func TestApplicationConfig_Env(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("環境を取得できる", func(t *testing.T) {
			t.Parallel()
			app := MockConfigForTest(t).app
			assert.Equal(t, expectedApplicationEnv, app.Env())
		})
	})
}

func TestApplicationConfig_Name(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("名前を取得できる", func(t *testing.T) {
			t.Parallel()
			app := MockConfigForTest(t).app
			assert.Equal(t, expectedApplicationName, app.Name())
		})
	})
}

func TestApplicationConfig_Mode(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("モードを取得できる", func(t *testing.T) {
			t.Parallel()
			app := MockConfigForTest(t).app
			assert.Equal(t, expectedApplicationMode, app.Mode())
		})
	})
}

func TestApplicationConfig_LogLevel(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("ログレベルを取得できる", func(t *testing.T) {
			t.Parallel()
			app := MockConfigForTest(t).app
			assert.Equal(t, expectedApplicationLogLevel, app.LogLevel())
		})
	})
}

func TestApplicationConfig_ShutdownTimeout(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("シャットダウンタイムアウトを取得できる", func(t *testing.T) {
			t.Parallel()
			app := MockConfigForTest(t).app
			assert.Equal(t, expectedAppShutdownTimeout, app.ShutdownTimeout())
		})
	})
}

func TestDatabaseConfig_Driver(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("ドライバーを取得できる", func(t *testing.T) {
			t.Parallel()
			database := MockConfigForTest(t).database
			assert.Equal(t, expectedDBDriver, database.Driver())
		})
	})
}

func TestDatabaseConfig_Host(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("ホストを取得できる", func(t *testing.T) {
			t.Parallel()
			database := MockConfigForTest(t).database
			assert.Equal(t, expectedDBHost, database.Host())
		})
	})
}

func TestDatabaseConfig_Port(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("ポートを取得できる", func(t *testing.T) {
			t.Parallel()
			database := MockConfigForTest(t).database
			assert.Equal(t, expectedDBPort, database.Port())
		})
	})
}

func TestDatabaseConfig_User(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("ユーザーを取得できる", func(t *testing.T) {
			t.Parallel()
			database := MockConfigForTest(t).database
			assert.Equal(t, expectedDBUser, database.User())
		})
	})
}

func TestDatabaseConfig_Password(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("パスワードを取得できる", func(t *testing.T) {
			t.Parallel()
			database := MockConfigForTest(t).database
			assert.Equal(t, expectedDBPassword, database.Password())
		})
	})
}

func TestDatabaseConfig_DBName(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("データベース名を取得できる", func(t *testing.T) {
			t.Parallel()
			database := MockConfigForTest(t).database
			assert.Equal(t, expectedDBName, database.DBName())
		})
	})
}

func TestDatabaseConfig_SSLMode(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("SSLモードを取得できる", func(t *testing.T) {
			t.Parallel()
			database := MockConfigForTest(t).database
			assert.Equal(t, expectedDBSSLMode, database.SSLMode())
		})
	})
}

func TestDatabaseConfig_PingTimeout(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("Pingタイムアウトを取得できる", func(t *testing.T) {
			t.Parallel()
			database := MockConfigForTest(t).database
			assert.Equal(t, expectedDBPingTimeout, database.PingTimeout())
		})
	})
}

func TestDatabaseConfig_SlowQueryWarnThreshold(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("スロークエリ警告閾値を取得できる", func(t *testing.T) {
			t.Parallel()
			database := MockConfigForTest(t).database
			assert.Equal(t, expectedDBSlowQueryWarnThreshold, database.SlowQueryWarnThreshold())
		})
	})
}

func TestDatabaseConfig_StatementTimeout(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("statement_timeout を取得できる", func(t *testing.T) {
			t.Parallel()
			database := MockConfigForTest(t).database
			assert.Equal(t, expectedDBStatementTimeout, database.StatementTimeout())
		})
	})
}

func TestDatabaseConfig_LockTimeout(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("lock_timeout を取得できる", func(t *testing.T) {
			t.Parallel()
			database := MockConfigForTest(t).database
			assert.Equal(t, expectedDBLockTimeout, database.LockTimeout())
		})
	})
}

func TestDatabaseConfig_TxMaxRetries(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("トランザクションリトライ最大試行回数を取得できる", func(t *testing.T) {
			t.Parallel()
			database := MockConfigForTest(t).database
			assert.Equal(t, expectedDBTxMaxRetries, database.TxMaxRetries())
		})
	})
}

func TestDatabaseConfig_TxRetryBaseBackoff(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("トランザクションリトライbackoff初期値を取得できる", func(t *testing.T) {
			t.Parallel()
			database := MockConfigForTest(t).database
			assert.Equal(t, expectedDBTxRetryBaseBackoff, database.TxRetryBaseBackoff())
		})
	})
}

func TestDatabaseConfig_TxRetryMaxBackoff(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("トランザクションリトライbackoff上限値を取得できる", func(t *testing.T) {
			t.Parallel()
			database := MockConfigForTest(t).database
			assert.Equal(t, expectedDBTxRetryMaxBackoff, database.TxRetryMaxBackoff())
		})
	})
}

func TestDBConnectionConfig_MaxConns(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("最大コネクション数を取得できる", func(t *testing.T) {
			t.Parallel()
			connection := MockConfigForTest(t).dbconnection
			assert.Equal(t, expectedDBMaxConnsInt32, connection.MaxConns())
		})
	})
}

func TestDBConnectionConfig_MinConns(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("最小コネクション数を取得できる", func(t *testing.T) {
			t.Parallel()
			connection := MockConfigForTest(t).dbconnection
			assert.Equal(t, expectedDBMinConnsInt32, connection.MinConns())
		})
	})
}

func TestDBConnectionConfig_MaxLifetime(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("最大ライフタイムを取得できる", func(t *testing.T) {
			t.Parallel()
			connection := MockConfigForTest(t).dbconnection
			assert.Equal(t, expectedDBMaxLifetime, connection.MaxLifetime())
		})
	})
}

func TestDBConnectionConfig_MaxIdleTime(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("最大アイドル時間を取得できる", func(t *testing.T) {
			t.Parallel()
			connection := MockConfigForTest(t).dbconnection
			assert.Equal(t, expectedDBMaxIdleTime, connection.MaxIdleTime())
		})
	})
}

func TestSecurityConfig_AllowedOrigins(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("許可オリジンを取得できる", func(t *testing.T) {
			t.Parallel()
			security := MockConfigForTest(t).security
			assert.Equal(t, strings.Split(expectedAllowedOrigins, ","), security.AllowedOrigins())
		})
	})
}

func TestSecurityConfig_ContentTypeNosniff(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("ContentTypeNosniffを取得できる", func(t *testing.T) {
			t.Parallel()
			security := MockConfigForTest(t).security
			assert.Equal(t, expectedContentTypeNosniff, security.ContentTypeNosniff())
		})
	})
}

func TestSecurityConfig_XFrameOptions(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("XFrameOptionsを取得できる", func(t *testing.T) {
			t.Parallel()
			security := MockConfigForTest(t).security
			assert.Equal(t, expectedXFrameOptions, security.XFrameOptions())
		})
	})
}

func TestSecurityConfig_HSTSMaxAge(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("HSTS最大期間を取得できる", func(t *testing.T) {
			t.Parallel()
			security := MockConfigForTest(t).security
			assert.Equal(t, expectedHSTSMaxAge, security.HSTSMaxAge())
		})
	})
}

func TestSecurityConfig_HSTSExcludeSubdomains(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("HSTSサブドメイン除外設定を取得できる", func(t *testing.T) {
			t.Parallel()
			security := MockConfigForTest(t).security
			assert.Equal(t, expectedHSTSExcludeSubdomains, security.HSTSExcludeSubdomains())
		})
	})
}

func TestSecurityConfig_HSTSPreloadEnabled(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("HSTSプリロード有効設定を取得できる", func(t *testing.T) {
			t.Parallel()
			security := MockConfigForTest(t).security
			assert.Equal(t, expectedHSTSPreloadEnabled, security.HSTSPreloadEnabled())
		})
	})
}

func TestSecurityConfig_ReferrerPolicy(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("リファラーポリシーを取得できる", func(t *testing.T) {
			t.Parallel()
			security := MockConfigForTest(t).security
			assert.Equal(t, expectedReferrerPolicy, security.ReferrerPolicy())
		})
	})
}

func TestSecurityConfig_BcryptCost(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("bcryptコストを取得できる", func(t *testing.T) {
			t.Parallel()
			security := MockConfigForTest(t).security
			assert.Equal(t, expectedBcryptCost, security.BcryptCost())
		})
	})
}

func TestSecureCookieConfig_SameSite(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("SameSite属性を取得できる", func(t *testing.T) {
			t.Parallel()
			secureCookie := MockConfigForTest(t).secureCookie
			assert.Equal(t, expectedSecureCookieSameSite, secureCookie.SameSite())
		})
	})
}

func TestSecureCookieConfig_Domain(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("ドメインを取得できる", func(t *testing.T) {
			t.Parallel()
			secureCookie := MockConfigForTest(t).secureCookie
			assert.Equal(t, expectedSecureCookieDomain, secureCookie.Domain())
		})
	})
}

func TestWorkerConfig_Concurrency(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("並行実行数を取得できる", func(t *testing.T) {
			t.Parallel()
			worker := MockConfigForTest(t).worker
			assert.Equal(t, expectedWorkerConcurrency, worker.Concurrency())
		})
	})
}

func TestWorkerConfig_MaxInFlight(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("最大インフライト数を取得できる", func(t *testing.T) {
			t.Parallel()
			worker := MockConfigForTest(t).worker
			assert.Equal(t, expectedWorkerMaxInFlight, worker.MaxInFlight())
		})
	})
}

func TestWorkerConfig_BatchSize(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("バッチサイズを取得できる", func(t *testing.T) {
			t.Parallel()
			worker := MockConfigForTest(t).worker
			assert.Equal(t, expectedWorkerBatchSize, worker.BatchSize())
		})
	})
}

func TestWorkerConfig_ExtendInterval(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("Extend周期を取得できる", func(t *testing.T) {
			t.Parallel()
			worker := MockConfigForTest(t).worker
			assert.Equal(t, expectedWorkerExtendInterval, worker.ExtendInterval())
		})
	})
}

func TestWorkerConfig_DrainTimeout(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("ドレインタイムアウトを取得できる", func(t *testing.T) {
			t.Parallel()
			worker := MockConfigForTest(t).worker
			assert.Equal(t, expectedWorkerDrainTimeout, worker.DrainTimeout())
		})
	})
}

func TestWorkerConfig_ReceiveCountWarnThreshold(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("再配送回数の警告閾値を取得できる", func(t *testing.T) {
			t.Parallel()
			worker := MockConfigForTest(t).worker
			assert.Equal(t, expectedWorkerReceiveCountWarnThreshold, worker.ReceiveCountWarnThreshold())
		})
	})
}

func TestWorkerConfig_CircuitFailureThreshold(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("サーキット失敗閾値を取得できる", func(t *testing.T) {
			t.Parallel()
			worker := MockConfigForTest(t).worker
			assert.Equal(t, expectedWorkerCircuitFailureThreshold, worker.CircuitFailureThreshold())
		})
	})
}

func TestWorkerConfig_CircuitOpenBackoffInitial(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("サーキットOpen初回backoffを取得できる", func(t *testing.T) {
			t.Parallel()
			worker := MockConfigForTest(t).worker
			assert.Equal(t, expectedWorkerCircuitOpenBackoffInitial, worker.CircuitOpenBackoffInitial())
		})
	})
}

func TestWorkerConfig_CircuitOpenBackoffMax(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("サーキットOpen backoff上限を取得できる", func(t *testing.T) {
			t.Parallel()
			worker := MockConfigForTest(t).worker
			assert.Equal(t, expectedWorkerCircuitOpenBackoffMax, worker.CircuitOpenBackoffMax())
		})
	})
}

func TestWorkerConfig_CircuitHalfOpenProbe(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("Half-open試行数を取得できる", func(t *testing.T) {
			t.Parallel()
			worker := MockConfigForTest(t).worker
			assert.Equal(t, expectedWorkerCircuitHalfOpenProbe, worker.CircuitHalfOpenProbe())
		})
	})
}

func TestWorkerConfig_HealthListenAddr(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("health listener待ち受けアドレスを取得できる", func(t *testing.T) {
			t.Parallel()
			worker := MockConfigForTest(t).worker
			assert.Equal(t, expectedWorkerHealthListenAddr, worker.HealthListenAddr())
		})
	})
}

func TestWorkerConfig_ProgressStaleAfter(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("進捗停滞判定時間を取得できる", func(t *testing.T) {
			t.Parallel()
			worker := MockConfigForTest(t).worker
			assert.Equal(t, expectedWorkerProgressStaleAfter, worker.ProgressStaleAfter())
		})
	})
}

func TestWorkerConfig_NackBackoffInitial(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("Nack初回backoffを取得できる", func(t *testing.T) {
			t.Parallel()
			worker := MockConfigForTest(t).worker
			assert.Equal(t, expectedWorkerNackBackoffInitial, worker.NackBackoffInitial())
		})
	})
}

func TestWorkerConfig_NackBackoffMax(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("Nack backoff上限を取得できる", func(t *testing.T) {
			t.Parallel()
			worker := MockConfigForTest(t).worker
			assert.Equal(t, expectedWorkerNackBackoffMax, worker.NackBackoffMax())
		})
	})
}

func TestOutboxConfig_Endpoint(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("エンドポイントを取得できる", func(t *testing.T) {
			t.Parallel()
			outbox := MockConfigForTest(t).outbox
			assert.Equal(t, expectedOutboxEndpoint, outbox.Endpoint())
		})
	})
}

func TestOutboxConfig_PollInterval(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("ポーリング間隔を取得できる", func(t *testing.T) {
			t.Parallel()
			outbox := MockConfigForTest(t).outbox
			assert.Equal(t, expectedOutboxPollInterval, outbox.PollInterval())
		})
	})
}

func TestOutboxConfig_ErrorBackoff(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("エラーbackoffを取得できる", func(t *testing.T) {
			t.Parallel()
			outbox := MockConfigForTest(t).outbox
			assert.Equal(t, expectedOutboxErrorBackoff, outbox.ErrorBackoff())
		})
	})
}

func TestOutboxConfig_BatchSize(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("バッチサイズを取得できる", func(t *testing.T) {
			t.Parallel()
			outbox := MockConfigForTest(t).outbox
			assert.Equal(t, expectedOutboxBatchSize, outbox.BatchSize())
		})
	})
}

func TestAuthConfig_Issuer(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("issuerを取得できる", func(t *testing.T) {
			t.Parallel()
			auth := MockConfigForTest(t).auth
			assert.Equal(t, expectedAuthIssuer, auth.Issuer())
		})
	})
}

func TestAuthConfig_Audience(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("audienceを取得できる", func(t *testing.T) {
			t.Parallel()
			auth := MockConfigForTest(t).auth
			assert.Equal(t, expectedAuthAudience, auth.Audience())
		})
	})
}

func TestAuthConfig_JWKSURL(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("JWKS URLを取得できる", func(t *testing.T) {
			t.Parallel()
			auth := MockConfigForTest(t).auth
			assert.Equal(t, expectedAuthJWKSURL, auth.JWKSURL())
		})
	})
}

func TestAuthConfig_AllowedAlgorithms(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("許可アルゴリズムを取得できる", func(t *testing.T) {
			t.Parallel()
			auth := MockConfigForTest(t).auth
			assert.Equal(t, expectedAuthAllowedAlgorithms, auth.AllowedAlgorithms())
		})
	})
}

func TestAuthConfig_ClockSkew(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("クロックスキューを取得できる", func(t *testing.T) {
			t.Parallel()
			auth := MockConfigForTest(t).auth
			assert.Equal(t, expectedAuthClockSkew, auth.ClockSkew())
		})
	})
}

func TestAuthConfig_JWKSCacheTTL(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()
		t.Run("JWKSキャッシュTTLを取得できる", func(t *testing.T) {
			t.Parallel()
			auth := MockConfigForTest(t).auth
			assert.Equal(t, expectedAuthJWKSCacheTTL, auth.JWKSCacheTTL())
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
		t.Run("CIDRを取得できる", func(t *testing.T) {
			t.Parallel()
			security := MockConfigForTest(t).security
			assert.Equal(t, expectedCIDR, security.CIDR())
		})

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
		t.Run("Secure属性を取得できる", func(t *testing.T) {
			t.Parallel()
			secureCookie := MockConfigForTest(t).secureCookie
			assert.Equal(t, expectedSecureCookieSecure, secureCookie.Secure())
		})

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

		t.Run("いずれかの exporter が有効値なら全体有効を取得できる", func(t *testing.T) {
			t.Parallel()
			observability := MockConfigForTest(t).observability
			assert.True(t, observability.Enabled())
		})

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
