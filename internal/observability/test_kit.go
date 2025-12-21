package observability

import (
	"context"
	"testing"

	"boilerplate-go/internal/logging"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const (
	tracer = "noop-tracer"
	layer  = "noop-layer"
	pkg    = "noop-pkg"
	span   = "noop-span"
)

// NewNoopTracerFactory は、テスト用に TracerFactory を無効化して返します。
func NewNoopTracerFactory(t *testing.T) TracerFactory {
	t.Helper()
	lf := logging.NewTestLogFieldBuilder(t)

	tp := noop.NewTracerProvider()
	tl := logging.NewTestLogger(t)
	return NewTracerFactory(tp, tl, lf)
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

func NewMockInfraLayerTracer(t *testing.T) LayerTracer {
	t.Helper()
	tf := NewNoopTracerFactory(t)
	return tf.Infra()
}

// NewNoopLayerTracer は、テスト用に LayerTracer を無効化して返します。
func NewNoopLayerTracer(t *testing.T) LayerTracer {
	t.Helper()
	return LayerTracer{
		log:     logging.NewTestLogger(t),
		lf:      logging.NewTestLogFieldBuilder(t),
		tracer:  noop.NewTracerProvider().Tracer(tracer),
		layer:   layer,
		pkgName: pkg,
	}
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
