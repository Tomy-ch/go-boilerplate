package observability

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const (
	tracer           = "noop-tracer"
	layer  layerName = "noop-layer"
	pkg              = "noop-pkg"
	span             = "noop-span"
)

// NewNoopTracerFactory は、テスト用に TracerFactory を無効化して返します。
func NewNoopTracerFactory(t *testing.T) TracerFactory {
	t.Helper()

	tp := noop.NewTracerProvider()
	return NewTracerFactory(tp)
}

// NewMockControllerLayerTracer は、テスト用のコントローラーレイヤートレーサーを生成します。
func NewMockControllerLayerTracer(t *testing.T) LayerTracer {
	t.Helper()
	tf := NewNoopTracerFactory(t)
	return tf.Controller()
}

// NewMockUsecaseLayerTracer は、テスト用のユースケースレイヤートレーサーを生成します。
func NewMockUsecaseLayerTracer(t *testing.T) LayerTracer {
	t.Helper()
	tf := NewNoopTracerFactory(t)
	return tf.Usecase()
}

// NewMockInfraLayerTracer は、テスト用のインフラレイヤートレーサーを生成します。
func NewMockInfraLayerTracer(t *testing.T) LayerTracer {
	t.Helper()
	tf := NewNoopTracerFactory(t)
	return tf.Infra()
}

// NewNoopLayerTracer は、テスト用に LayerTracer を無効化して返します。
func NewNoopLayerTracer(t *testing.T) LayerTracer {
	t.Helper()
	return LayerTracer{
		tracer:  noop.NewTracerProvider().Tracer(tracer),
		layer:   layer,
		pkgName: pkg,
	}
}

// NewNoopWorkerMetrics は、テスト用に no-op の MeterProvider から WorkerMetrics を生成します。
func NewNoopWorkerMetrics(t *testing.T) *WorkerMetrics {
	t.Helper()
	wm, err := NewWorkerMetrics(metricnoop.NewMeterProvider())
	if err != nil {
		t.Fatalf("failed to build noop worker metrics: %v", err)
	}
	return wm
}

// NewNoopHTTPClientMetrics は、テスト用に no-op の MeterProvider から HTTPClientMetrics を生成します。
func NewNoopHTTPClientMetrics(t *testing.T) *HTTPClientMetrics {
	t.Helper()
	hm, err := NewHTTPClientMetrics(metricnoop.NewMeterProvider())
	if err != nil {
		t.Fatalf("failed to build noop http client metrics: %v", err)
	}
	return hm
}

// NewNoopHTTPClientTransport は、テスト用に no-op TracerProvider と実 propagator から HTTPClientTransport を
// 生成します。SSRF ガードは無効化（loopback/httptest 宛てを許可）します。
func NewNoopHTTPClientTransport(t *testing.T) *HTTPClientTransport {
	t.Helper()
	return newHTTPClientTransport(noop.NewTracerProvider(), NewTextMapPropagator(), permissiveDialControl)
}

// NewStubSpanContext は、テスト用のスタブSpanコンテキストを返します。
func NewStubSpanContext(t *testing.T) (context.Context, func()) {
	t.Helper()

	prev := otel.GetTracerProvider()

	tp := sdktrace.NewTracerProvider()
	tr := tp.Tracer(tracer)

	ctx, span := tr.Start(context.Background(), span)

	return ctx, func() {
		span.End()
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(prev)
	}
}
