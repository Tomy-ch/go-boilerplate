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

// metricPresent は、指定名の metric が収集結果に存在する（＝データ点が 1 つ以上 emit された）かを返します。
func metricPresent(rm metricdata.ResourceMetrics, name string) bool {
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				return true
			}
		}
	}
	return false
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

// newIdempotencyMetricsForTest は、収集器付きの IdempotencyMetrics を生成します。
func newIdempotencyMetricsForTest(t *testing.T) (*observability.IdempotencyMetrics, *sdkmetric.ManualReader) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	im, err := observability.NewIdempotencyMetrics(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	require.NoError(t, err)
	return im, reader
}

// collectIdempotencyMetrics は、reader から収集結果を取り出します。
func collectIdempotencyMetrics(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	return rm
}

func TestIdempotencyMetrics_IncHit(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("idempotency.requests へ result=hit と operation_id を付与して計上する", func(t *testing.T) {
			t.Parallel()

			im, reader := newIdempotencyMetricsForTest(t)

			im.IncHit(context.Background(), "PostResources")

			rm := collectIdempotencyMetrics(t, reader)
			assert.Equal(t, []string{"idempotency.requests"}, metricNamesOf(t, rm))
			assert.Equal(t, "hit", attributeOf(t, rm, "idempotency.requests", "result"))
			assert.Equal(t, "PostResources", operationIDOf(t, rm, "idempotency.requests"))
			assert.Equal(t, int64(1), counterValueOf(t, rm, "idempotency.requests"))
		})
	})
}

func TestIdempotencyMetrics_IncMiss(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("idempotency.requests へ result=miss と operation_id を付与して計上する", func(t *testing.T) {
			t.Parallel()

			im, reader := newIdempotencyMetricsForTest(t)

			im.IncMiss(context.Background(), "PostResources")

			rm := collectIdempotencyMetrics(t, reader)
			assert.Equal(t, []string{"idempotency.requests"}, metricNamesOf(t, rm))
			assert.Equal(t, "miss", attributeOf(t, rm, "idempotency.requests", "result"))
			assert.Equal(t, "PostResources", operationIDOf(t, rm, "idempotency.requests"))
		})
	})
}

func TestIdempotencyMetrics_IncConflict(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("idempotency.requests へ result=conflict と operation_id を付与して計上する", func(t *testing.T) {
			t.Parallel()

			im, reader := newIdempotencyMetricsForTest(t)

			im.IncConflict(context.Background(), "PostResources")

			rm := collectIdempotencyMetrics(t, reader)
			assert.Equal(t, []string{"idempotency.requests"}, metricNamesOf(t, rm))
			assert.Equal(t, "conflict", attributeOf(t, rm, "idempotency.requests", "result"))
			assert.Equal(t, "PostResources", operationIDOf(t, rm, "idempotency.requests"))
		})
	})
}

func TestIdempotencyMetrics_IncFingerprintMismatch(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("idempotency.requests へ result=fingerprint_mismatch と operation_id を付与して計上する", func(t *testing.T) {
			t.Parallel()

			im, reader := newIdempotencyMetricsForTest(t)

			im.IncFingerprintMismatch(context.Background(), "PostResources")

			rm := collectIdempotencyMetrics(t, reader)
			assert.Equal(t, []string{"idempotency.requests"}, metricNamesOf(t, rm))
			assert.Equal(t, "fingerprint_mismatch", attributeOf(t, rm, "idempotency.requests", "result"))
			assert.Equal(t, "PostResources", operationIDOf(t, rm, "idempotency.requests"))
		})
	})
}

func TestIdempotencyMetrics_IncClaimFailure(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("idempotency.failures へ phase=claim と operation_id を付与して計上する", func(t *testing.T) {
			t.Parallel()

			im, reader := newIdempotencyMetricsForTest(t)

			im.IncClaimFailure(context.Background(), "PostResources")

			rm := collectIdempotencyMetrics(t, reader)
			// 判定結果の requests ではなく内部失敗の failures へ計上される。
			assert.Equal(t, []string{"idempotency.failures"}, metricNamesOf(t, rm))
			assert.Equal(t, "claim", attributeOf(t, rm, "idempotency.failures", "phase"))
			assert.Equal(t, "PostResources", operationIDOf(t, rm, "idempotency.failures"))
		})
	})
}

func TestIdempotencyMetrics_IncCompleteFailure(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("idempotency.failures へ phase=complete と operation_id を付与して計上する", func(t *testing.T) {
			t.Parallel()

			im, reader := newIdempotencyMetricsForTest(t)

			im.IncCompleteFailure(context.Background(), "PostResources")

			rm := collectIdempotencyMetrics(t, reader)
			assert.Equal(t, []string{"idempotency.failures"}, metricNamesOf(t, rm))
			assert.Equal(t, "complete", attributeOf(t, rm, "idempotency.failures", "phase"))
			assert.Equal(t, "PostResources", operationIDOf(t, rm, "idempotency.failures"))
		})
	})
}

func TestIdempotencyMetrics_IncExpiredCleanup(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("idempotency.expired_cleanup へ job ラベル付きで削除件数を加算する", func(t *testing.T) {
			t.Parallel()

			im, reader := newIdempotencyMetricsForTest(t)

			// 固定値 1 ではなく削除件数 count を加算する。
			im.IncExpiredCleanup(context.Background(), 42)

			rm := collectIdempotencyMetrics(t, reader)
			assert.Equal(t, []string{"idempotency.expired_cleanup"}, metricNamesOf(t, rm))
			assert.Equal(t, int64(42), counterValueOf(t, rm, "idempotency.expired_cleanup"))
			assert.Equal(t, "idempotency_gc", attributeOf(t, rm, "idempotency.expired_cleanup", "job"))
		})
	})
}

func TestIdempotencyMetrics_IncExpiredCleanupFailure(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("専用カウンタへ job=idempotency_gc で計上しper-requestのfailuresへは計上しない", func(t *testing.T) {
			t.Parallel()

			im, reader := newIdempotencyMetricsForTest(t)

			im.IncExpiredCleanupFailure(context.Background())

			rm := collectIdempotencyMetrics(t, reader)
			// GC 失敗は per-request の failures ではなく専用カウンタへ計上され、
			// operation_id="unknown" を帯びず job ラベルで成功と対称になる。
			assert.Equal(t, "idempotency_gc", attributeOf(t, rm, "idempotency.expired_cleanup_failure", "job"))
			assert.Equal(t, int64(1), counterValueOf(t, rm, "idempotency.expired_cleanup_failure"))
			// 併せて per-request の idempotency.failures には一切 emit されないことを固定する
			// （GC 失敗を専用カウンタと failures の両方へ計上する二重計上の回帰を検出する）。
			assert.False(t, metricPresent(rm, "idempotency.failures"),
				"GC 失敗が per-request failures へ二重計上されないこと")
		})
	})
}
