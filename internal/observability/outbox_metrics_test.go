package observability_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/embedded"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"go-boilerplate/internal/observability"
)

// failingGaugeMeterProvider は、Int64Gauge 生成に失敗する Meter を返す MeterProvider スタブです。
// NewOutboxMetrics は最初に Int64Gauge(lag_seconds) を呼ぶため、ここでエラーを注入すれば生成失敗経路を通せます。
type failingGaugeMeterProvider struct{ embedded.MeterProvider }

// failingGaugeMeter は、Int64Gauge 生成のみエラーを返す Meter スタブです。
type failingGaugeMeter struct{ metric.Meter }

func (failingGaugeMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter {
	return failingGaugeMeter{}
}

func (failingGaugeMeter) Int64Gauge(string, ...metric.Int64GaugeOption) (metric.Int64Gauge, error) {
	return nil, errMeter
}

func TestNewOutboxMetrics(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("outbox 固有の全 metric が登録され記録できる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			om, err := observability.NewOutboxMetrics(provider)
			require.NoError(t, err)

			om.SetLagSeconds(ctx, 42)
			om.IncDead(ctx)

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(ctx, &rm))

			names := map[string]bool{}
			for _, sm := range rm.ScopeMetrics {
				for _, m := range sm.Metrics {
					names[m.Name] = true
				}
			}

			for _, want := range []string{"outbox.lag_seconds", "outbox.dead"} {
				assert.Contains(t, names, want)
			}
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("計装生成に失敗した場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			om, err := observability.NewOutboxMetrics(failingGaugeMeterProvider{})

			require.ErrorIs(t, err, errMeter)
			assert.Nil(t, om)
		})
	})
}

// metricByName は、ResourceMetrics から指定名の metric を 1 件取り出します。
func metricByName(t *testing.T, rm metricdata.ResourceMetrics, name string) metricdata.Metrics {
	t.Helper()
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				return m
			}
		}
	}
	t.Fatalf("metric %s not found", name)
	return metricdata.Metrics{}
}

func TestOutboxMetrics_IncDead(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("outbox.dead を Sum[int64] として 1 加算する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			om, err := observability.NewOutboxMetrics(provider)
			require.NoError(t, err)

			om.IncDead(ctx)

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(ctx, &rm))

			// Sum[int64] への型アサートで、型誤実装（Gauge 化等）を検出可能にする。
			s, ok := metricByName(t, rm, "outbox.dead").Data.(metricdata.Sum[int64])
			require.True(t, ok)
			require.NotEmpty(t, s.DataPoints)
			assert.Equal(t, int64(1), s.DataPoints[0].Value)
			// lag gauge 側へ取り違えて計上していないこと。
			assert.False(t, metricPresent(rm, "outbox.lag_seconds"))
		})

		t.Run("複数回の計上は累積する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			om, err := observability.NewOutboxMetrics(provider)
			require.NoError(t, err)

			om.IncDead(ctx)
			om.IncDead(ctx)
			om.IncDead(ctx)

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(ctx, &rm))

			assert.Equal(t, int64(3), counterValueOf(t, rm, "outbox.dead"))
		})
	})
}

func TestOutboxMetrics_SetLagSeconds(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("outbox.lag_seconds を Gauge[int64] として値を記録する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			om, err := observability.NewOutboxMetrics(provider)
			require.NoError(t, err)

			om.SetLagSeconds(ctx, 42)

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(ctx, &rm))

			// Gauge[int64] への型アサートで、型誤実装（Sum 化等）を検出可能にする。
			g, ok := metricByName(t, rm, "outbox.lag_seconds").Data.(metricdata.Gauge[int64])
			require.True(t, ok)
			require.NotEmpty(t, g.DataPoints)
			assert.Equal(t, int64(42), g.DataPoints[0].Value)
			// dead counter 側へ取り違えて計上していないこと。
			assert.False(t, metricPresent(rm, "outbox.dead"))
		})

		t.Run("後の記録で上書きされ累積しない", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			om, err := observability.NewOutboxMetrics(provider)
			require.NoError(t, err)

			om.SetLagSeconds(ctx, 42)
			// pending 無しは 0 を記録する運用のため、直近値へ落ちることを固定する。
			om.SetLagSeconds(ctx, 0)

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(ctx, &rm))

			g, ok := metricByName(t, rm, "outbox.lag_seconds").Data.(metricdata.Gauge[int64])
			require.True(t, ok)
			require.NotEmpty(t, g.DataPoints)
			assert.Equal(t, int64(0), g.DataPoints[0].Value)
		})
	})
}
