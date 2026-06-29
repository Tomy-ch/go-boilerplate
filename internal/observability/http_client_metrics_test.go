package observability_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-boilerplate/internal/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
}

func TestHTTPClientMetricsRecord(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("各計装メソッドがパニックせず計上できる", func(t *testing.T) {
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
	})
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
