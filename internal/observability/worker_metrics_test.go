package observability_test

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/embedded"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/xerrors"
)

// errMeter は、failingMeter が Int64Counter 生成で返す注入エラーです。
var errMeter = xerrors.New("int64 counter creation failed")

// errHistogram は、histogramFailingMeter が Float64Histogram 生成で返す注入エラーです。
var errHistogram = xerrors.New("float64 histogram creation failed")

// errUpDownCounter は、upDownCounterFailingMeter が Int64UpDownCounter 生成で返す注入エラーです。
var errUpDownCounter = xerrors.New("int64 up down counter creation failed")

// failingMeterProvider は、Int64Counter 生成に失敗する Meter を返す MeterProvider スタブです。
type failingMeterProvider struct{ embedded.MeterProvider }

// failingMeter は、Int64Counter 生成のみエラーを返す Meter スタブです。
// NewWorkerMetrics は最初に Int64Counter を呼ぶため、ここでエラーを注入すれば生成失敗経路を通せます。
type failingMeter struct{ metric.Meter }

// histogramFailingMeterProvider は、Float64Histogram 生成のみに失敗する Meter を返す MeterProvider スタブです。
type histogramFailingMeterProvider struct{ embedded.MeterProvider }

// histogramFailingMeter は、Counter 生成は no-op へ委譲しつつ Float64Histogram のみエラーを返す Meter スタブです。
// NewWorkerMetrics は Counter 群の後に Histogram を生成するため、histogram 生成失敗経路を選択的に通せます。
type histogramFailingMeter struct{ metric.Meter }

// upDownCounterFailingMeterProvider は、Int64UpDownCounter 生成のみに失敗する Meter を返す MeterProvider スタブです。
type upDownCounterFailingMeterProvider struct{ embedded.MeterProvider }

// upDownCounterFailingMeter は、Counter/Histogram 生成は no-op へ委譲しつつ Int64UpDownCounter のみエラーを返す Meter スタブです。
// NewWorkerMetrics は最後に UpDownCounter を生成するため、その生成失敗経路を選択的に通せます。
type upDownCounterFailingMeter struct{ metric.Meter }

func (failingMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter { return failingMeter{} }

func (failingMeter) Int64Counter(string, ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	return nil, errMeter
}

func (histogramFailingMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter {
	return histogramFailingMeter{Meter: metricnoop.NewMeterProvider().Meter("")}
}

func (histogramFailingMeter) Float64Histogram(
	string, ...metric.Float64HistogramOption,
) (metric.Float64Histogram, error) {
	return nil, errHistogram
}

func (upDownCounterFailingMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter {
	return upDownCounterFailingMeter{Meter: metricnoop.NewMeterProvider().Meter("")}
}

func (upDownCounterFailingMeter) Int64UpDownCounter(
	string, ...metric.Int64UpDownCounterOption,
) (metric.Int64UpDownCounter, error) {
	return nil, errUpDownCounter
}

func TestNewWorkerMetrics(t *testing.T) {
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

			// メソッドごとに区別できる測定値を与える（manual reader は測定のある計装のみ出力）。
			wm.Received(ctx, 5)
			wm.Processed(ctx)
			wm.Failed(ctx)
			wm.Retried(ctx)
			wm.DLQ(ctx)
			wm.PollError(ctx)
			wm.ExtendError(ctx)
			wm.RecordLatencyMs(ctx, 12.5)
			wm.InFlightAdd(ctx, 3)

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

			// 記録値そのものを検証する（名前存在のみの疑似陽性を防ぐ）。
			assert.Equal(t, int64(5), counterValueOf(t, rm, "worker.received"))
			assert.Equal(t, int64(1), counterValueOf(t, rm, "worker.processed"))
			assert.Equal(t, int64(3), counterValueOf(t, rm, "worker.in_flight"))
			assert.InDelta(t, 12.5, histogramSumOf(t, rm, "worker.processing_latency_ms"), 1e-9)
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

		t.Run("histogram 生成に失敗した場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			wm, err := observability.NewWorkerMetrics(histogramFailingMeterProvider{})

			require.ErrorIs(t, err, errHistogram)
			assert.Nil(t, wm)
		})

		t.Run("upDownCounter 生成に失敗した場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			wm, err := observability.NewWorkerMetrics(upDownCounterFailingMeterProvider{})

			require.ErrorIs(t, err, errUpDownCounter)
			assert.Nil(t, wm)
		})
	})
}

// histogramSumOf は、指定 histogram の最初のデータ点の Sum を返します。
func histogramSumOf(t *testing.T, rm metricdata.ResourceMetrics, name string) float64 {
	t.Helper()
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			h, ok := m.Data.(metricdata.Histogram[float64])
			require.True(t, ok)
			require.NotEmpty(t, h.DataPoints)
			return h.DataPoints[0].Sum
		}
	}
	t.Fatalf("metric %s not found", name)
	return 0
}

// metricNamesOf は、収集結果に現れた metric 名を昇順で返します。
// manual reader は測定のあった計装のみ出力するため、「どの計装へ計上されたか」を排他的に固定できます。
func metricNamesOf(t *testing.T, rm metricdata.ResourceMetrics) []string {
	t.Helper()
	var names []string
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			names = append(names, m.Name)
		}
	}
	sort.Strings(names)
	return names
}

// newWorkerMetricsForTest は、収集器付きの WorkerMetrics を生成します。
func newWorkerMetricsForTest(t *testing.T) (*observability.WorkerMetrics, *sdkmetric.ManualReader) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	wm, err := observability.NewWorkerMetrics(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	require.NoError(t, err)
	return wm, reader
}

// collectWorkerMetrics は、reader から収集結果を取り出します。
func collectWorkerMetrics(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	return rm
}

func TestWorkerMetrics_Received(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("worker.received のみへ引数の件数を加算する", func(t *testing.T) {
			t.Parallel()

			wm, reader := newWorkerMetricsForTest(t)

			// 固定値 1 ではなく引数 n を加算することを、1 以外の値で固定する。
			wm.Received(context.Background(), 5)

			rm := collectWorkerMetrics(t, reader)
			assert.Equal(t, []string{"worker.received"}, metricNamesOf(t, rm))
			assert.Equal(t, int64(5), counterValueOf(t, rm, "worker.received"))
		})
	})
}

func TestWorkerMetrics_Processed(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("worker.processed のみへ 1 を加算する", func(t *testing.T) {
			t.Parallel()

			wm, reader := newWorkerMetricsForTest(t)

			wm.Processed(context.Background())

			rm := collectWorkerMetrics(t, reader)
			assert.Equal(t, []string{"worker.processed"}, metricNamesOf(t, rm))
			assert.Equal(t, int64(1), counterValueOf(t, rm, "worker.processed"))
		})
	})
}

func TestWorkerMetrics_Failed(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("worker.failed のみへ 1 を加算する", func(t *testing.T) {
			t.Parallel()

			wm, reader := newWorkerMetricsForTest(t)

			wm.Failed(context.Background())

			rm := collectWorkerMetrics(t, reader)
			assert.Equal(t, []string{"worker.failed"}, metricNamesOf(t, rm))
			assert.Equal(t, int64(1), counterValueOf(t, rm, "worker.failed"))
		})
	})
}

func TestWorkerMetrics_Retried(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("worker.retried のみへ 1 を加算する", func(t *testing.T) {
			t.Parallel()

			wm, reader := newWorkerMetricsForTest(t)

			wm.Retried(context.Background())

			rm := collectWorkerMetrics(t, reader)
			assert.Equal(t, []string{"worker.retried"}, metricNamesOf(t, rm))
			assert.Equal(t, int64(1), counterValueOf(t, rm, "worker.retried"))
		})
	})
}

func TestWorkerMetrics_DLQ(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("worker.dlq のみへ 1 を加算する", func(t *testing.T) {
			t.Parallel()

			wm, reader := newWorkerMetricsForTest(t)

			wm.DLQ(context.Background())

			rm := collectWorkerMetrics(t, reader)
			assert.Equal(t, []string{"worker.dlq"}, metricNamesOf(t, rm))
			assert.Equal(t, int64(1), counterValueOf(t, rm, "worker.dlq"))
		})
	})
}

func TestWorkerMetrics_PollError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("worker.poll_errors のみへ 1 を加算する", func(t *testing.T) {
			t.Parallel()

			wm, reader := newWorkerMetricsForTest(t)

			wm.PollError(context.Background())

			rm := collectWorkerMetrics(t, reader)
			assert.Equal(t, []string{"worker.poll_errors"}, metricNamesOf(t, rm))
			assert.Equal(t, int64(1), counterValueOf(t, rm, "worker.poll_errors"))
		})
	})
}

func TestWorkerMetrics_ExtendError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("worker.extend_errors のみへ 1 を加算する", func(t *testing.T) {
			t.Parallel()

			wm, reader := newWorkerMetricsForTest(t)

			wm.ExtendError(context.Background())

			rm := collectWorkerMetrics(t, reader)
			assert.Equal(t, []string{"worker.extend_errors"}, metricNamesOf(t, rm))
			assert.Equal(t, int64(1), counterValueOf(t, rm, "worker.extend_errors"))
		})
	})
}

func TestWorkerMetrics_RecordLatencyMs(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("worker.processing_latency_ms のみへ処理時間を histogram として記録する", func(t *testing.T) {
			t.Parallel()

			wm, reader := newWorkerMetricsForTest(t)

			wm.RecordLatencyMs(context.Background(), 12.5)
			wm.RecordLatencyMs(context.Background(), 7.5)

			rm := collectWorkerMetrics(t, reader)
			assert.Equal(t, []string{"worker.processing_latency_ms"}, metricNamesOf(t, rm))
			// 2 回の記録が 1 つの分布へ集約される（Sum=20.0 / Count=2）。
			assert.InDelta(t, 20.0, histogramSumOf(t, rm, "worker.processing_latency_ms"), 1e-9)
		})
	})
}

func TestWorkerMetrics_InFlightAdd(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("worker.in_flight のみへ delta を加算する", func(t *testing.T) {
			t.Parallel()

			wm, reader := newWorkerMetricsForTest(t)

			wm.InFlightAdd(context.Background(), 3)

			rm := collectWorkerMetrics(t, reader)
			assert.Equal(t, []string{"worker.in_flight"}, metricNamesOf(t, rm))
			assert.Equal(t, int64(3), counterValueOf(t, rm, "worker.in_flight"))
		})

		t.Run("負の delta で減算できる", func(t *testing.T) {
			t.Parallel()

			wm, reader := newWorkerMetricsForTest(t)

			// 単調増加の counter ではなく UpDownCounter であること（処理中数は減る）。
			wm.InFlightAdd(context.Background(), 3)
			wm.InFlightAdd(context.Background(), -2)

			assert.Equal(t, int64(1), counterValueOf(t, collectWorkerMetrics(t, reader), "worker.in_flight"))
		})
	})
}
