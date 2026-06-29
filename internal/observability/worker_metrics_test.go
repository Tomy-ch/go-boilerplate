package observability_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/embedded"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"go-boilerplate/internal/observability"
)

// errMeter は、failingMeter が Int64Counter 生成で返す注入エラーです。
var errMeter = errors.New("int64 counter creation failed")

// failingMeterProvider は、Int64Counter 生成に失敗する Meter を返す MeterProvider スタブです。
type failingMeterProvider struct{ embedded.MeterProvider }

// failingMeter は、Int64Counter 生成のみエラーを返す Meter スタブです。
// NewWorkerMetrics は最初に Int64Counter を呼ぶため、ここでエラーを注入すれば生成失敗経路を通せます。
type failingMeter struct{ metric.Meter }

func (failingMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter { return failingMeter{} }

func (failingMeter) Int64Counter(string, ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	return nil, errMeter
}

func Test_NewWorkerMetrics_D2(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("engine 所有の全 metric が登録され記録できる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			wm, err := observability.NewWorkerMetrics(provider)
			require.NoError(t, err)

			// 各計装に 1 度ずつ測定値を入れる（manual reader は測定のある計装のみ出力するため）。
			wm.Received(ctx, 1)
			wm.Processed(ctx)
			wm.Failed(ctx)
			wm.Retried(ctx)
			wm.DLQ(ctx)
			wm.PollError(ctx)
			wm.ExtendError(ctx)
			wm.RecordLatencyMs(ctx, 1)
			wm.InFlightAdd(ctx, 1)

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
				"worker.dlq", "worker.poll_errors", "worker.extend_errors",
				"worker.processing_latency_ms", "worker.in_flight",
			} {
				assert.Contains(t, names, want)
			}
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("計装生成に失敗した場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			wm, err := observability.NewWorkerMetrics(failingMeterProvider{})

			require.ErrorIs(t, err, errMeter)
			assert.Nil(t, wm)
		})
	})
}
