package config

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigTestingSetters(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
	t.Run("正常系", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
		cfg := MockConfigForTest(t)

		t.Run("アプリケーションモードを設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			expected := "test-mode"
			cfg.app.SetApplicationMode(t, expected)
			assert.Equal(t, expected, cfg.app.Mode())
		})

		t.Run("アプリケーション環境を設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			expected := "test-env"
			cfg.app.SetApplicationEnv(t, expected)
			assert.Equal(t, expected, cfg.app.Env())
		})

		t.Run("アプリケーションログレベルを設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			expected := "warn"
			cfg.app.SetApplicationLogLevel(t, expected)
			assert.Equal(t, expected, cfg.app.LogLevel())
		})

		t.Run("サーバーポートを設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			expected := 8081
			cfg.server.SetServerPort(t, expected)
			assert.Equal(t, expected, cfg.server.Port())
		})

		t.Run("DBクエリ引数のマスク設定を設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			expected := true
			cfg.observability.SetObservabilityMaskedDBQueryArgs(t, expected)
			assert.Equal(t, expected, cfg.observability.MaskedDBQueryArgs())
		})

		t.Run("trace exporter を設定して有効判定が変わる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			cfg.observability.SetObservabilityTracesExporter(t, "")
			assert.False(t, cfg.observability.TracesEnabled())
		})

		t.Run("metric exporter を設定して有効判定が変わる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			cfg.observability.SetObservabilityMetricsExporter(t, "")
			assert.False(t, cfg.observability.MetricsEnabled())
		})

		t.Run("log exporter を設定して有効判定が変わる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			cfg.observability.SetObservabilityLogsExporter(t, "")
			assert.False(t, cfg.observability.LogsEnabled())
		})

		t.Run("OTLPプロトコルを設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			expected := "grpc"
			cfg.observability.SetObservabilityOTLPProtocol(t, expected)
			assert.Equal(t, expected, cfg.observability.OTLPProtocol())
		})

		t.Run("OTLPエンドポイントを設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			expected := "http://test-collector:4318"
			cfg.observability.SetObservabilityOTLPEndpoint(t, expected)
			assert.Equal(t, expected, cfg.observability.OTLPEndpoint())
		})

		t.Run("メトリクスポートを設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			expected := 16060
			cfg.metrics.SetMetricsPort(t, expected)
			assert.Equal(t, expected, cfg.metrics.Port())
		})

		t.Run("データベースホストを設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			expected := "test-host"
			cfg.database.SetDatabaseHost(t, expected)
			assert.Equal(t, expected, cfg.database.Host())
		})

		t.Run("データベース名を設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			expected := "test-name"
			cfg.database.SetDatabaseName(t, expected)
			assert.Equal(t, expected, cfg.database.DBName())
		})

		t.Run("最大コネクション数を設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			expected := int32(20)
			cfg.dbconnection.SetMaxConns(t, expected)
			assert.Equal(t, expected, cfg.dbconnection.MaxConns())
		})

		t.Run("CIDRを設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			_, testCIDR, err := net.ParseCIDR("192.168.1.0/24")
			require.NoError(t, err)
			cfg.security.SetCIDR(t, testCIDR)
			assert.Equal(t, testCIDR, cfg.security.CIDR())
		})

		t.Run("outboxのbatch sizeを設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			expected := 7
			cfg.outbox.SetOutboxBatchSize(t, expected)
			assert.Equal(t, expected, cfg.outbox.BatchSize())
		})

		t.Run("outboxのエンドポイントを設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			expected := "http://test-relay:8080"
			cfg.outbox.SetOutboxEndpoint(t, expected)
			assert.Equal(t, expected, cfg.outbox.Endpoint())
		})

		t.Run("outboxのpoll間隔を設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			expected := 3 * time.Second
			cfg.outbox.SetOutboxPollInterval(t, expected)
			assert.Equal(t, expected, cfg.outbox.PollInterval())
		})

		t.Run("outboxのエラー時待機時間を設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			expected := 9 * time.Second
			cfg.outbox.SetOutboxErrorBackoff(t, expected)
			assert.Equal(t, expected, cfg.outbox.ErrorBackoff())
		})

		t.Run("worker health listener アドレスを設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			expected := "127.0.0.1:0"
			cfg.worker.SetHealthListenAddr(t, expected)
			assert.Equal(t, expected, cfg.worker.HealthListenAddr())
		})

		t.Run("セキュアクッキーの SameSite を設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			expected := "Lax"
			cfg.secureCookie.SetSameSite(t, expected)
			assert.Equal(t, expected, cfg.secureCookie.SameSite())
		})

		t.Run("セキュアクッキーの Domain を設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			expected := "example.test"
			cfg.secureCookie.SetDomain(t, expected)
			assert.Equal(t, expected, cfg.secureCookie.Domain())
		})

		t.Run("クリーンアップ後に元の値へ復元される", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			t.Run("サーバーポートが復元される", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
				original := cfg.server.Port()

				t.Run("一時的にポートを上書きする", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
					cfg.server.SetServerPort(t, original+1)
					assert.Equal(t, original+1, cfg.server.Port())
				})

				// 内側サブテスト終了時に Cleanup が発火し、元値へ戻る。
				assert.Equal(t, original, cfg.server.Port())
			})
		})
	})
}
