package observability_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
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

			im.IncHit(ctx, "PostResources")
			im.IncMiss(ctx, "PostResources")
			im.IncConflict(ctx, "PostResources")
			im.IncFingerprintMismatch(ctx, "PostResources")
			im.IncClaimFailure(ctx, "PostResources")
			im.IncCompleteFailure(ctx, "PostResources")
			im.IncExpiredCleanup(ctx, 7)
			im.IncExpiredCleanupFailure(ctx)

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
				"idempotency.expired_cleanup_failure",
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

			im.IncHit(ctx, "")

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

			im.IncExpiredCleanup(ctx, 42)

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(ctx, &rm))

			assert.Equal(t, int64(42), counterValueOf(t, rm, "idempotency.expired_cleanup"))
		})

		t.Run("判定系メソッドは result ラベルへ対応する値を emit する", func(t *testing.T) {
			t.Parallel()

			t.Run("IncHit は result=hit", func(t *testing.T) {
				t.Parallel()

				ctx := context.Background()
				reader := sdkmetric.NewManualReader()
				provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

				im, err := observability.NewIdempotencyMetrics(provider)
				require.NoError(t, err)

				im.IncHit(ctx, "PostResources")

				var rm metricdata.ResourceMetrics
				require.NoError(t, reader.Collect(ctx, &rm))

				assert.Equal(t, "hit", attributeOf(t, rm, "idempotency.requests", "result"))
			})

			t.Run("IncMiss は result=miss", func(t *testing.T) {
				t.Parallel()

				ctx := context.Background()
				reader := sdkmetric.NewManualReader()
				provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

				im, err := observability.NewIdempotencyMetrics(provider)
				require.NoError(t, err)

				im.IncMiss(ctx, "PostResources")

				var rm metricdata.ResourceMetrics
				require.NoError(t, reader.Collect(ctx, &rm))

				assert.Equal(t, "miss", attributeOf(t, rm, "idempotency.requests", "result"))
			})

			t.Run("IncConflict は result=conflict", func(t *testing.T) {
				t.Parallel()

				ctx := context.Background()
				reader := sdkmetric.NewManualReader()
				provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

				im, err := observability.NewIdempotencyMetrics(provider)
				require.NoError(t, err)

				im.IncConflict(ctx, "PostResources")

				var rm metricdata.ResourceMetrics
				require.NoError(t, reader.Collect(ctx, &rm))

				assert.Equal(t, "conflict", attributeOf(t, rm, "idempotency.requests", "result"))
			})

			t.Run("IncFingerprintMismatch は result=fingerprint_mismatch", func(t *testing.T) {
				t.Parallel()

				ctx := context.Background()
				reader := sdkmetric.NewManualReader()
				provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

				im, err := observability.NewIdempotencyMetrics(provider)
				require.NoError(t, err)

				im.IncFingerprintMismatch(ctx, "PostResources")

				var rm metricdata.ResourceMetrics
				require.NoError(t, reader.Collect(ctx, &rm))

				assert.Equal(t, "fingerprint_mismatch", attributeOf(t, rm, "idempotency.requests", "result"))
			})
		})

		t.Run("失敗系メソッドは phase ラベルへ対応する値を emit する", func(t *testing.T) {
			t.Parallel()

			t.Run("IncClaimFailure は phase=claim", func(t *testing.T) {
				t.Parallel()

				ctx := context.Background()
				reader := sdkmetric.NewManualReader()
				provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

				im, err := observability.NewIdempotencyMetrics(provider)
				require.NoError(t, err)

				im.IncClaimFailure(ctx, "PostResources")

				var rm metricdata.ResourceMetrics
				require.NoError(t, reader.Collect(ctx, &rm))

				assert.Equal(t, "claim", attributeOf(t, rm, "idempotency.failures", "phase"))
			})

			t.Run("IncCompleteFailure は phase=complete", func(t *testing.T) {
				t.Parallel()

				ctx := context.Background()
				reader := sdkmetric.NewManualReader()
				provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

				im, err := observability.NewIdempotencyMetrics(provider)
				require.NoError(t, err)

				im.IncCompleteFailure(ctx, "PostResources")

				var rm metricdata.ResourceMetrics
				require.NoError(t, reader.Collect(ctx, &rm))

				assert.Equal(t, "complete", attributeOf(t, rm, "idempotency.failures", "phase"))
			})
		})

		t.Run("IncExpiredCleanupFailure は専用カウンタに job=idempotency_gc を emit する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			im, err := observability.NewIdempotencyMetrics(provider)
			require.NoError(t, err)

			im.IncExpiredCleanupFailure(ctx)

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(ctx, &rm))

			// GC 失敗は per-request の failures ではなく専用カウンタへ計上され、
			// operation_id="unknown" を帯びず job ラベルで成功と対称になる。
			assert.Equal(t, "idempotency_gc", attributeOf(t, rm, "idempotency.expired_cleanup_failure", "job"))
			assert.Equal(t, int64(1), counterValueOf(t, rm, "idempotency.expired_cleanup_failure"))
		})

		t.Run("IncExpiredCleanup は job=idempotency_gc を emit する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			im, err := observability.NewIdempotencyMetrics(provider)
			require.NoError(t, err)

			im.IncExpiredCleanup(ctx, 1)

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(ctx, &rm))

			assert.Equal(t, "idempotency_gc", attributeOf(t, rm, "idempotency.expired_cleanup", "job"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("計装生成に失敗した場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			im, err := observability.NewIdempotencyMetrics(failingMeterProvider{})

			require.ErrorIs(t, err, errMeter)
			assert.Nil(t, im)
		})
	})
}

// attributeOf は、指定 counter の最初のデータ点から指定キーの属性値を取り出します。
func attributeOf(t *testing.T, rm metricdata.ResourceMetrics, name, key string) string {
	t.Helper()
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok)
			require.NotEmpty(t, sum.DataPoints)
			v, ok := sum.DataPoints[0].Attributes.Value(attribute.Key(key))
			require.True(t, ok)
			return v.AsString()
		}
	}
	t.Fatalf("metric %s not found", name)
	return ""
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
