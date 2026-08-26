package observability_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func TestNewHTTPClientMetrics(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("MeterProviderから計装一式を生成できる", func(t *testing.T) {
			t.Parallel()

			hm, err := observability.NewHTTPClientMetrics(metricnoop.NewMeterProvider())
			require.NoError(t, err)
			assert.NotNil(t, hm)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("計装生成に失敗した場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			hm, err := observability.NewHTTPClientMetrics(failingMeterProvider{})

			require.ErrorIs(t, err, errMeter)
			assert.Nil(t, hm)
		})
	})
}

// newHTTPClientMetricsForTest は、収集器付きの HTTPClientMetrics を生成します。
func newHTTPClientMetricsForTest(t *testing.T) (*observability.HTTPClientMetrics, *sdkmetric.ManualReader) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	hm, err := observability.NewHTTPClientMetrics(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	require.NoError(t, err)
	return hm, reader
}

// collectHTTPClientMetrics は、reader から収集結果を取り出します。
func collectHTTPClientMetrics(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	return rm
}

// gaugeValueOf は、指定 gauge の最初のデータ点の値を返します。
func gaugeValueOf(t *testing.T, rm metricdata.ResourceMetrics, name string) int64 {
	t.Helper()
	g, ok := metricByName(t, rm, name).Data.(metricdata.Gauge[int64])
	require.True(t, ok)
	require.NotEmpty(t, g.DataPoints)
	return g.DataPoints[0].Value
}

// attrOfAny は、Sum / Histogram / Gauge いずれの metric でも最初のデータ点から指定キーの属性値を取り出します。
func attrOfAny(t *testing.T, rm metricdata.ResourceMetrics, name, key string) string {
	t.Helper()
	var attrs attribute.Set
	switch d := metricByName(t, rm, name).Data.(type) {
	case metricdata.Sum[int64]:
		require.NotEmpty(t, d.DataPoints)
		attrs = d.DataPoints[0].Attributes
	case metricdata.Histogram[float64]:
		require.NotEmpty(t, d.DataPoints)
		attrs = d.DataPoints[0].Attributes
	case metricdata.Gauge[int64]:
		require.NotEmpty(t, d.DataPoints)
		attrs = d.DataPoints[0].Attributes
	default:
		t.Fatalf("unsupported metric data type for %s", name)
	}
	v, ok := attrs.Value(attribute.Key(key))
	require.True(t, ok)
	return v.AsString()
}

func TestNewHTTPClientTransport(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成したRoundTripperでリクエストを送信できる", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(srv.Close)

			transport := observability.NewHTTPClientTransport(
				tracenoop.NewTracerProvider(), observability.NewTextMapPropagator(),
			)
			require.NotNil(t, transport)

			// httptest は loopback のため、SSRF ガードを通すには private 許可フラグが要る。
			ctx := observability.ContextWithAllowPrivateNetwork(context.Background(), true)
			client := &http.Client{Transport: transport.RoundTripper()}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
			require.NoError(t, err)

			resp, err := client.Do(req)
			require.NoError(t, err)
			t.Cleanup(func() { _ = resp.Body.Close() })
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	})
}

func TestHTTPClientMetrics_RecordRequest(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("httpclient.requests へ downstream と status_class を付与して計上する", func(t *testing.T) {
			t.Parallel()

			hm, reader := newHTTPClientMetricsForTest(t)

			hm.RecordRequest(context.Background(), "outbox", "2xx")

			rm := collectHTTPClientMetrics(t, reader)
			assert.Equal(t, []string{"httpclient.requests"}, metricNamesOf(t, rm))
			assert.Equal(t, int64(1), counterValueOf(t, rm, "httpclient.requests"))
			assert.Equal(t, "outbox", attributeOf(t, rm, "httpclient.requests", "downstream"))
			assert.Equal(t, "2xx", attributeOf(t, rm, "httpclient.requests", "status_class"))
		})

		t.Run("status_class 違いは別系列として計上する", func(t *testing.T) {
			t.Parallel()

			hm, reader := newHTTPClientMetricsForTest(t)
			ctx := context.Background()

			hm.RecordRequest(ctx, "outbox", "2xx")
			hm.RecordRequest(ctx, "outbox", "5xx")

			rm := collectHTTPClientMetrics(t, reader)
			assert.Equal(t, int64(2), counterValueOf(t, rm, "httpclient.requests"))
			sum, ok := metricByName(t, rm, "httpclient.requests").Data.(metricdata.Sum[int64])
			require.True(t, ok)
			assert.Len(t, sum.DataPoints, 2)
		})
	})
}

func TestHTTPClientMetrics_RecordError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("httpclient.errors へ downstream と reason を付与して計上する", func(t *testing.T) {
			t.Parallel()

			hm, reader := newHTTPClientMetricsForTest(t)

			hm.RecordError(context.Background(), "outbox", "transport")

			rm := collectHTTPClientMetrics(t, reader)
			assert.Equal(t, []string{"httpclient.errors"}, metricNamesOf(t, rm))
			assert.Equal(t, int64(1), counterValueOf(t, rm, "httpclient.errors"))
			assert.Equal(t, "outbox", attributeOf(t, rm, "httpclient.errors", "downstream"))
			assert.Equal(t, "transport", attributeOf(t, rm, "httpclient.errors", "reason"))
		})
	})
}

func TestHTTPClientMetrics_RecordRetry(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("httpclient.retries へ downstream を付与して計上する", func(t *testing.T) {
			t.Parallel()

			hm, reader := newHTTPClientMetricsForTest(t)

			hm.RecordRetry(context.Background(), "outbox")

			rm := collectHTTPClientMetrics(t, reader)
			assert.Equal(t, []string{"httpclient.retries"}, metricNamesOf(t, rm))
			assert.Equal(t, int64(1), counterValueOf(t, rm, "httpclient.retries"))
			assert.Equal(t, "outbox", attributeOf(t, rm, "httpclient.retries", "downstream"))
		})

		t.Run("複数回のリトライは累積する", func(t *testing.T) {
			t.Parallel()

			hm, reader := newHTTPClientMetricsForTest(t)
			ctx := context.Background()

			hm.RecordRetry(ctx, "outbox")
			hm.RecordRetry(ctx, "outbox")

			assert.Equal(t, int64(2), counterValueOf(t, collectHTTPClientMetrics(t, reader), "httpclient.retries"))
		})
	})
}

func TestHTTPClientMetrics_RecordLatencyMs(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("httpclient.request_latency_ms へ downstream を付与して所要時間を記録する", func(t *testing.T) {
			t.Parallel()

			hm, reader := newHTTPClientMetricsForTest(t)

			hm.RecordLatencyMs(context.Background(), "outbox", 12.5)

			rm := collectHTTPClientMetrics(t, reader)
			assert.Equal(t, []string{"httpclient.request_latency_ms"}, metricNamesOf(t, rm))
			assert.InDelta(t, 12.5, histogramSumOf(t, rm, "httpclient.request_latency_ms"), 1e-9)
			assert.Equal(t, "outbox", attrOfAny(t, rm, "httpclient.request_latency_ms", "downstream"))
		})
	})
}

func TestHTTPClientMetrics_InFlightAdd(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("httpclient.in_flight へ downstream を付与して delta を加算する", func(t *testing.T) {
			t.Parallel()

			hm, reader := newHTTPClientMetricsForTest(t)

			hm.InFlightAdd(context.Background(), "outbox", 3)

			rm := collectHTTPClientMetrics(t, reader)
			assert.Equal(t, []string{"httpclient.in_flight"}, metricNamesOf(t, rm))
			assert.Equal(t, int64(3), counterValueOf(t, rm, "httpclient.in_flight"))
			assert.Equal(t, "outbox", attributeOf(t, rm, "httpclient.in_flight", "downstream"))
		})

		t.Run("負の delta で減算できる", func(t *testing.T) {
			t.Parallel()

			hm, reader := newHTTPClientMetricsForTest(t)
			ctx := context.Background()

			// 単調増加の counter ではなく UpDownCounter であること（完了時に減る）。
			hm.InFlightAdd(ctx, "outbox", 3)
			hm.InFlightAdd(ctx, "outbox", -1)

			assert.Equal(t, int64(2), counterValueOf(t, collectHTTPClientMetrics(t, reader), "httpclient.in_flight"))
		})
	})
}

func TestHTTPClientMetrics_SetBreakerState(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("httpclient.breaker_state へ downstream を付与して状態値を記録する", func(t *testing.T) {
			t.Parallel()

			hm, reader := newHTTPClientMetricsForTest(t)

			hm.SetBreakerState(context.Background(), "outbox", 2)

			rm := collectHTTPClientMetrics(t, reader)
			assert.Equal(t, []string{"httpclient.breaker_state"}, metricNamesOf(t, rm))
			assert.Equal(t, int64(2), gaugeValueOf(t, rm, "httpclient.breaker_state"))
			assert.Equal(t, "outbox", attrOfAny(t, rm, "httpclient.breaker_state", "downstream"))
		})

		t.Run("後の記録で状態が上書きされ累積しない", func(t *testing.T) {
			t.Parallel()

			hm, reader := newHTTPClientMetricsForTest(t)
			ctx := context.Background()

			// gauge のため open(2) → closed(0) の復帰が直近値として反映される。
			hm.SetBreakerState(ctx, "outbox", 2)
			hm.SetBreakerState(ctx, "outbox", 0)

			assert.Equal(t, int64(0), gaugeValueOf(t, collectHTTPClientMetrics(t, reader), "httpclient.breaker_state"))
		})
	})
}
