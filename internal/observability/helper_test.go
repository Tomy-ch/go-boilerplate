package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"
)

func TestShouldLogWithSpan(t *testing.T) {
	t.Parallel()
	t.Run("オブザーバビリティ有効かつスパンあり -> true", func(t *testing.T) {
		t.Parallel()

		cfg := config.MockConfigForTest(t)
		obsCfg := config.NewObservabilityConfig(cfg)

		tp := sdktrace.NewTracerProvider()
		tracer := tp.Tracer("test")
		ctx, sp := tracer.Start(context.Background(), "s")
		defer func() { sp.End(); _ = tp.Shutdown(context.Background()) }()

		got := ShouldLogWithSpan(ctx, obsCfg)
		assert.True(t, got)
	})

	t.Run("オブザーバビリティ有効だがスパン無し -> false", func(t *testing.T) {
		t.Parallel()

		cfg := config.MockConfigForTest(t)
		obsCfg := config.NewObservabilityConfig(cfg)

		ctx := context.Background()
		got := ShouldLogWithSpan(ctx, obsCfg)
		assert.False(t, got)
	})

	t.Run("オブザーバビリティ無効だがスパンあり -> false", func(t *testing.T) {
		t.Parallel()

		obsCfg := &config.ObservabilityConfig{}

		tp := sdktrace.NewTracerProvider()
		tracer := tp.Tracer("test")
		ctx, sp := tracer.Start(context.Background(), "s")
		defer func() { sp.End(); _ = tp.Shutdown(context.Background()) }()

		got := ShouldLogWithSpan(ctx, obsCfg)
		assert.False(t, got)
	})

	t.Run("オブザーバビリティ無効かつスパン無し -> false", func(t *testing.T) {
		t.Parallel()

		obsCfg := &config.ObservabilityConfig{}
		ctx := context.Background()
		got := ShouldLogWithSpan(ctx, obsCfg)
		assert.False(t, got)
	})
}

func TestBuildSpanName(t *testing.T) {
	t.Parallel()

	layer := "layer"
	pkgName := "mypkg"
	funcName := "MyFunc"

	expected := "layer.mypkg.MyFunc"
	actual := BuildSpanName(layer, pkgName, funcName)
	assert.Equal(t, expected, actual)
}

func TestExtractTraceContext(t *testing.T) {
	t.Run("Contextにスパン情報がない場合、空のTraceContextが返る", func(t *testing.T) {
		expected := TraceContext{}

		ctx := context.Background()
		actual := ExtractTraceContext(ctx)
		assert.Equal(t, expected, actual)
	})

	t.Run("Contextにスパン情報がある場合、正しいTraceContextが返る", func(t *testing.T) {
		ctx, end := NewStubSpanContext(t)
		defer end()

		span := trace.SpanFromContext(ctx).SpanContext()
		assert.True(t, span.IsValid())

		expected := TraceContext{
			traceID: span.TraceID().String(),
			spanID:  span.SpanID().String(),
		}
		actual := ExtractTraceContext(ctx)
		assert.Equal(t, expected, actual)
	})
}

func TestStartSpanWithParent(t *testing.T) {
	t.Parallel()

	tp := sdktrace.NewTracerProvider()
	tracer := tp.Tracer("test")
	defer func() { _ = tp.Shutdown(context.Background()) }()

	layerTracer := LayerTracer{
		log:     logging.NewTestLogger(t),
		lf:      logging.NewTestLogFieldBuilder(t),
		tracer:  tracer,
		layer:   "test-layer",
		pkgName: "test-pkg",
	}

	parentCtx, parentSpan := tracer.Start(context.Background(), "parent-span")
	defer parentSpan.End()

	tc, childCtx, end := StartSpanWithParent(parentCtx, layerTracer, "child-span")
	defer end()

	childSpan := trace.SpanFromContext(childCtx)
	assert.True(t, childSpan.SpanContext().IsValid())

	assert.Equal(t, parentSpan.SpanContext().TraceID().String(), tc.TraceID())
	assert.Equal(t, parentSpan.SpanContext().SpanID().String(), tc.ParentSpanID())
	assert.Equal(t, childSpan.SpanContext().SpanID().String(), tc.SpanID())
}

func TestTraceContext_IDs(t *testing.T) {
	t.Parallel()

	traceID := "trace-id-123"
	spanID := "span-id-456"
	tc := TraceContext{
		traceID: traceID,
		spanID:  spanID,
	}

	assert.Equal(t, traceID, tc.TraceID())
	assert.Equal(t, spanID, tc.SpanID())
}
