package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// newIdempotencyMetricsWithReader は、収集器付きの IdempotencyMetrics を生成します。
func newIdempotencyMetricsWithReader(t *testing.T) (*IdempotencyMetrics, *sdkmetric.ManualReader) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	im, err := NewIdempotencyMetrics(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	require.NoError(t, err)
	return im, reader
}

// collectLabelValues は、reader を collect して metricName の counter が持つ labelKey の値を返します。
func collectLabelValues(t *testing.T, reader *sdkmetric.ManualReader, metricName, labelKey string) []string {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	values, ok := counterLabelValues(rm, metricName, labelKey)
	require.True(t, ok)
	return values
}

func Test_normalizeOperationID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非空の operationID はそのまま返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "CreateUser", normalizeOperationID("CreateUser"))
		})

		t.Run("空の operationID は unknown へ丸める", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "unknown", normalizeOperationID(""))
		})
	})
}

func TestIdempotencyMetrics_incRequest(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("requests カウンタへ operation_id と result を付与して計上する", func(t *testing.T) {
			t.Parallel()

			im, reader := newIdempotencyMetricsWithReader(t)

			im.incRequest(context.Background(), "PostResources", "hit")

			assert.Equal(t, []string{"PostResources"},
				collectLabelValues(t, reader, "idempotency.requests", "operation_id"))
			assert.Equal(t, []string{"hit"}, collectLabelValues(t, reader, "idempotency.requests", "result"))
			// 内部失敗の failures 側へは計上しない。
			assert.Empty(t, collectLabelValues(t, reader, "idempotency.failures", "phase"))
		})

		t.Run("空の operationID は unknown へ丸めて計上する", func(t *testing.T) {
			t.Parallel()

			im, reader := newIdempotencyMetricsWithReader(t)

			im.incRequest(context.Background(), "", "miss")

			assert.Equal(t, []string{"unknown"},
				collectLabelValues(t, reader, "idempotency.requests", "operation_id"))
		})

		t.Run("result が異なれば別系列として計上する", func(t *testing.T) {
			t.Parallel()

			im, reader := newIdempotencyMetricsWithReader(t)
			ctx := context.Background()

			im.incRequest(ctx, "PostResources", "hit")
			im.incRequest(ctx, "PostResources", "miss")

			assert.ElementsMatch(t, []string{"hit", "miss"},
				collectLabelValues(t, reader, "idempotency.requests", "result"))
		})
	})
}

func TestIdempotencyMetrics_incFailure(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("failures カウンタへ operation_id と phase を付与して計上する", func(t *testing.T) {
			t.Parallel()

			im, reader := newIdempotencyMetricsWithReader(t)

			im.incFailure(context.Background(), "PostResources", "claim")

			assert.Equal(t, []string{"PostResources"},
				collectLabelValues(t, reader, "idempotency.failures", "operation_id"))
			assert.Equal(t, []string{"claim"}, collectLabelValues(t, reader, "idempotency.failures", "phase"))
			// 判定結果の requests 側へは計上しない。
			assert.Empty(t, collectLabelValues(t, reader, "idempotency.requests", "result"))
		})

		t.Run("空の operationID は unknown へ丸めて計上する", func(t *testing.T) {
			t.Parallel()

			im, reader := newIdempotencyMetricsWithReader(t)

			im.incFailure(context.Background(), "", "complete")

			assert.Equal(t, []string{"unknown"},
				collectLabelValues(t, reader, "idempotency.failures", "operation_id"))
		})

		t.Run("phase が異なれば別系列として計上する", func(t *testing.T) {
			t.Parallel()

			im, reader := newIdempotencyMetricsWithReader(t)
			ctx := context.Background()

			im.incFailure(ctx, "PostResources", "claim")
			im.incFailure(ctx, "PostResources", "complete")

			assert.ElementsMatch(t, []string{"claim", "complete"},
				collectLabelValues(t, reader, "idempotency.failures", "phase"))
		})
	})
}
