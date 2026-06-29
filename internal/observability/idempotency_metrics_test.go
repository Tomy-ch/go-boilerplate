package observability_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"go-boilerplate/internal/observability"
)

func TestNewIdempotencyMetrics(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全 metric が登録され全メソッドが記録できる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			im, err := observability.NewIdempotencyMetrics(provider)
			require.NoError(t, err)

			im.IncHit("PostResources")
			im.IncMiss("PostResources")
			im.IncConflict("PostResources")
			im.IncFingerprintMismatch("PostResources")
			im.IncClaimFailure("PostResources")
			im.IncCompleteFailure("PostResources")
			im.IncExpiredCleanup(7)
			im.IncExpiredCleanupFailure()

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(ctx, &rm))

			names := map[string]bool{}
			for _, sm := range rm.ScopeMetrics {
				for _, m := range sm.Metrics {
					names[m.Name] = true
				}
			}

			for _, want := range []string{
				"idempotency.requests",
				"idempotency.failures",
				"idempotency.expired_cleanup",
			} {
				assert.Contains(t, names, want)
			}
		})

		t.Run("operationID が空なら result ラベルの operation_id は unknown になる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			im, err := observability.NewIdempotencyMetrics(provider)
			require.NoError(t, err)

			im.IncHit("")

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(ctx, &rm))

			gotOperationID := operationIDOf(t, rm, "idempotency.requests")
			assert.Equal(t, "unknown", gotOperationID)
		})

		t.Run("expired cleanup は削除件数を value として加算する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			im, err := observability.NewIdempotencyMetrics(provider)
			require.NoError(t, err)

			im.IncExpiredCleanup(42)

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(ctx, &rm))

			assert.Equal(t, int64(42), counterValueOf(t, rm, "idempotency.expired_cleanup"))
		})
	})
}

// operationIDOf は、指定 counter の最初のデータ点から operation_id 属性値を取り出します。
func operationIDOf(t *testing.T, rm metricdata.ResourceMetrics, name string) string {
	t.Helper()
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok)
			require.NotEmpty(t, sum.DataPoints)
			v, ok := sum.DataPoints[0].Attributes.Value("operation_id")
			require.True(t, ok)
			return v.AsString()
		}
	}
	t.Fatalf("metric %s not found", name)
	return ""
}

// counterValueOf は、指定 counter の全データ点の合計値を返します。
func counterValueOf(t *testing.T, rm metricdata.ResourceMetrics, name string) int64 {
	t.Helper()
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok)
			var total int64
			for _, dp := range sum.DataPoints {
				total += dp.Value
			}
			return total
		}
	}
	t.Fatalf("metric %s not found", name)
	return 0
}
