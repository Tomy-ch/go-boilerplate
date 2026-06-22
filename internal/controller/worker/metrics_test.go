package worker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func Test_newMetrics_D2(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("engine 所有の全 metric が登録され記録できる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			mt, err := newMetrics(provider.Meter("test"))
			require.NoError(t, err)

			// 各計装に 1 度ずつ測定値を入れる（manual reader は測定のある計装のみ出力するため）。
			mt.received.Add(ctx, 1)
			mt.processed.Add(ctx, 1)
			mt.failed.Add(ctx, 1)
			mt.retried.Add(ctx, 1)
			mt.dlq.Add(ctx, 1)
			mt.pollErrors.Add(ctx, 1)
			mt.latencyMs.Record(ctx, 1)
			mt.inFlight.Add(ctx, 1)

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(ctx, &rm))

			names := map[string]bool{}
			for _, sm := range rm.ScopeMetrics {
				for _, m := range sm.Metrics {
					names[m.Name] = true
				}
			}

			for _, want := range []string{
				"worker.received", "worker.processed", "worker.failed", "worker.retried",
				"worker.dlq", "worker.poll_errors", "worker.processing_latency_ms", "worker.in_flight",
			} {
				assert.Contains(t, names, want)
			}
		})
	})
}
