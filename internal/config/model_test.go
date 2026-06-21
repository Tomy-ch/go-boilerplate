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

		t.Run("認証設定のコンストラクタが内部フィールドへの参照を返す", func(t *testing.T) {
			t.Parallel()
			authCfg := NewAuthConfig(cfg)
			assert.Same(t, &cfg.auth, authCfg)
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

			t.Run("有効フラグを取得できる", func(t *testing.T) {
				t.Parallel()
				assert.True(t, observability.Enabled())
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

		t.Run("データベース設定", func(t *testing.T) {
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

		t.Run("認証設定", func(t *testing.T) {
			t.Parallel()
			auth := cfg.auth

			t.Run("Cookie名を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedAuthCookieName, auth.CookieName())
			})

			t.Run("ヘッダー名を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedAuthHeaderName, auth.HeaderName())
			})

			t.Run("Bearerヘッダー許可を取得できる", func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, expectedAuthAllowedHeaderBearer, auth.AllowedHeaderBearer())
			})
		})
	})
}

func Test_ApplicationConfig_IsProductionMode(t *testing.T) {
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

func Test_ApplicationConfig_IsDevelopmentMode(t *testing.T) {
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
