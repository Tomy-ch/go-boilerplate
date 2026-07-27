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

func TestHTTPClientMetricsRecord(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("noop provider では各計装メソッドがパニックしない", func(t *testing.T) {
			t.Parallel()

			hm := observability.NewNoopHTTPClientMetrics(t)
			ctx := context.Background()

			assert.NotPanics(t, func() {
				hm.RecordRequest(ctx, "sample", "2xx")
				hm.RecordError(ctx, "sample", "transport")
				hm.RecordRetry(ctx, "sample")
				hm.RecordLatencyMs(ctx, "sample", 12.3)
				hm.InFlightAdd(ctx, "sample", 1)
				hm.InFlightAdd(ctx, "sample", -1)
				hm.SetBreakerState(ctx, "sample", 2)
			})
		})
	})
}

func TestHTTPClientMetricsAttributes(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("RecordRequest は downstream と status_class を付与して計上する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			hm, err := observability.NewHTTPClientMetrics(provider)
			require.NoError(t, err)

			hm.RecordRequest(ctx, "outbox", "2xx")

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(ctx, &rm))

			assert.Equal(t, int64(1), counterValueOf(t, rm, "httpclient.requests"))
			assert.Equal(t, "outbox", attributeOf(t, rm, "httpclient.requests", "downstream"))
			assert.Equal(t, "2xx", attributeOf(t, rm, "httpclient.requests", "status_class"))
		})

		t.Run("RecordError は downstream と reason を付与して計上する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			hm, err := observability.NewHTTPClientMetrics(provider)
			require.NoError(t, err)

			hm.RecordError(ctx, "outbox", "transport")

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(ctx, &rm))

			assert.Equal(t, int64(1), counterValueOf(t, rm, "httpclient.errors"))
			assert.Equal(t, "outbox", attributeOf(t, rm, "httpclient.errors", "downstream"))
			assert.Equal(t, "transport", attributeOf(t, rm, "httpclient.errors", "reason"))
		})

		t.Run("RecordRetry は downstream を付与してリトライ回数を計上する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			hm, err := observability.NewHTTPClientMetrics(provider)
			require.NoError(t, err)

			hm.RecordRetry(ctx, "outbox")

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(ctx, &rm))

			assert.Equal(t, int64(1), counterValueOf(t, rm, "httpclient.retries"))
			assert.Equal(t, "outbox", attributeOf(t, rm, "httpclient.retries", "downstream"))
		})

		t.Run("RecordLatencyMs は downstream を付与して所要時間を記録する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			hm, err := observability.NewHTTPClientMetrics(provider)
			require.NoError(t, err)

			hm.RecordLatencyMs(ctx, "outbox", 12.5)

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(ctx, &rm))

			assert.InDelta(t, 12.5, histogramSumOf(t, rm, "httpclient.request_latency_ms"), 1e-9)
			assert.Equal(t, "outbox", attrOfAny(t, rm, "httpclient.request_latency_ms", "downstream"))
		})

		t.Run("InFlightAdd は downstream を付与して処理中数を増減する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			hm, err := observability.NewHTTPClientMetrics(provider)
			require.NoError(t, err)

			hm.InFlightAdd(ctx, "outbox", 3)
			hm.InFlightAdd(ctx, "outbox", -1)

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(ctx, &rm))

			assert.Equal(t, int64(2), counterValueOf(t, rm, "httpclient.in_flight"))
			assert.Equal(t, "outbox", attributeOf(t, rm, "httpclient.in_flight", "downstream"))
		})

		t.Run("SetBreakerState は downstream を付与して状態値を記録する", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

			hm, err := observability.NewHTTPClientMetrics(provider)
			require.NoError(t, err)

			hm.SetBreakerState(ctx, "outbox", 2)

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(ctx, &rm))

			assert.Equal(t, int64(2), gaugeValueOf(t, rm, "httpclient.breaker_state"))
			assert.Equal(t, "outbox", attrOfAny(t, rm, "httpclient.breaker_state", "downstream"))
		})
	})
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

func TestHTTPClientMetrics_InFlightAdd(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestHTTPClientMetrics_RecordError(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestHTTPClientMetrics_RecordLatencyMs(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestHTTPClientMetrics_RecordRequest(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestHTTPClientMetrics_RecordRetry(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func TestHTTPClientMetrics_SetBreakerState(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
