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

			transport := observability.NewHTTPClientTransport(tracenoop.NewTracerProvider())
			require.NotNil(t, transport)

			client := &http.Client{Transport: transport.RoundTripper()}
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
			require.NoError(t, err)

			resp, err := client.Do(req)
			require.NoError(t, err)
			t.Cleanup(func() { _ = resp.Body.Close() })
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	})
}
