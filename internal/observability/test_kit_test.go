package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewNoopTracerFactory(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Noop TracerProviderを保持するfactoryを返す", func(t *testing.T) {
			t.Parallel()
			tp := noop.NewTracerProvider()

			actual := NewNoopTracerFactory(t)
			tf, ok := actual.(*tracerFactory)
			require.True(t, ok)
			assert.Equal(t, tp, tf.tp)
		})
	})
}

func TestNewMockControllerLayerTracer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Controllerレイヤを設定したMockLayerTracerを返す", func(t *testing.T) {
			t.Parallel()
			actual := NewMockControllerLayerTracer(t)
			assert.Equal(t, Controller, actual.layer)
			assert.NotNil(t, actual.tracer)
		})
	})
}

func TestNewMockUsecaseLayerTracer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("UsecaseレイヤをセットしたMockLayerTracerを返す", func(t *testing.T) {
			t.Parallel()
			actual := NewMockUsecaseLayerTracer(t)
			assert.Equal(t, Usecase, actual.layer)
			assert.NotNil(t, actual.tracer)
		})
	})
}

func TestNewMockInfraLayerTracer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("InfraレイヤをセットしたMockLayerTracerを返す", func(t *testing.T) {
			t.Parallel()
			actual := NewMockInfraLayerTracer(t)
			assert.Equal(t, Infra, actual.layer)
			assert.NotNil(t, actual.tracer)
		})
	})
}

func TestNewNoopLayerTracer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("テスト用LayerTracerを返す", func(t *testing.T) {
			t.Parallel()
			actual := NewNoopLayerTracer(t)
			assert.Equal(t, layer, actual.layer)
			assert.Equal(t, pkg, actual.pkgName)
			assert.NotNil(t, actual.tracer)
		})
	})
}

func TestNewNoopWorkerMetrics(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("no-op MeterProvider から WorkerMetrics を生成する", func(t *testing.T) {
			t.Parallel()

			wm := NewNoopWorkerMetrics(t)
			require.NotNil(t, wm)
			assert.NotPanics(t, func() { wm.Processed(context.Background()) })
		})
	})
}

func TestNewNoopOutboxMetrics(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("no-op MeterProvider から OutboxMetrics を生成する", func(t *testing.T) {
			t.Parallel()

			om := NewNoopOutboxMetrics(t)
			assert.NotNil(t, om)
		})
	})
}

func TestNewNoopHTTPClientTransport(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("SSRFガードを無効化し loopback 宛ての実接続を許可する", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()

			transport := NewNoopHTTPClientTransport(t)
			require.NotNil(t, transport)

			// permissive な dial control により loopback(httptest) 宛ての実 dial が拒否されず接続できる。
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
			require.NoError(t, err)
			resp, err := (&http.Client{Transport: transport.RoundTripper()}).Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		})
	})
}

func TestNewStubSpanContext(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効なSpanContextを保持するcontextとspan終了関数を返す", func(t *testing.T) {
			t.Parallel()
			ctx, span := NewStubSpanContext(t)
			require.NotEmpty(t, ctx)
			require.NotNil(t, span)
			defer span()

			spanCtx := trace.SpanFromContext(ctx)
			assert.True(t, spanCtx.SpanContext().IsValid())
		})
	})
}
