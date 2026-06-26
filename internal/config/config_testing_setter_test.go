package config

import (
	"net"
	"testing"

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

		t.Run("認証ヘッダー名を設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			expected := "X-TEST-AUTH"
			cfg.auth.SetHeaderName(t, expected)
			assert.Equal(t, expected, cfg.auth.HeaderName())
		})

		t.Run("Bearerヘッダー許可を設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			expected := true
			cfg.auth.SetAllowedHeaderBearer(t, expected)
			assert.Equal(t, expected, cfg.auth.AllowedHeaderBearer())
		})

		t.Run("outboxのbatch sizeを設定して取得できる", func(t *testing.T) { //nolint:paralleltest // 共有状態のため並列化不可
			expected := 7
			cfg.outbox.SetOutboxBatchSize(t, expected)
			assert.Equal(t, expected, cfg.outbox.BatchSize())
		})
	})
}
