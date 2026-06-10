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

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("オブザーバビリティ有効かつスパンありの場合、trueを返す", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			obsCfg := config.NewObservabilityConfig(cfg)

			tp := sdktrace.NewTracerProvider()
			tracer := tp.Tracer("test")
			ctx, sp := tracer.Start(context.Background(), "s")
			defer func() { sp.End(); _ = tp.Shutdown(context.Background()) }()

			assert.True(t, ShouldLogWithSpan(ctx, obsCfg))
		})

		t.Run("オブザーバビリティ有効だがスパン無しの場合、falseを返す", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			obsCfg := config.NewObservabilityConfig(cfg)

			assert.False(t, ShouldLogWithSpan(context.Background(), obsCfg))
		})

		t.Run("オブザーバビリティ無効だがスパンありの場合、falseを返す", func(t *testing.T) {
			t.Parallel()

			obsCfg := &config.ObservabilityConfig{}

			tp := sdktrace.NewTracerProvider()
			tracer := tp.Tracer("test")
			ctx, sp := tracer.Start(context.Background(), "s")
			defer func() { sp.End(); _ = tp.Shutdown(context.Background()) }()

			assert.False(t, ShouldLogWithSpan(ctx, obsCfg))
		})

		t.Run("オブザーバビリティ無効かつスパン無しの場合、falseを返す", func(t *testing.T) {
			t.Parallel()

			obsCfg := &config.ObservabilityConfig{}
			assert.False(t, ShouldLogWithSpan(context.Background(), obsCfg))
		})
	})
}

func TestBuildSpanName(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("layer.pkg.func形式のスパン名を組み立てる", func(t *testing.T) {
			t.Parallel()

			expected := "layer.mypkg.MyFunc"
			actual := BuildSpanName("layer", "mypkg", "MyFunc")
			assert.Equal(t, expected, actual)
		})
	})
}

func TestExtractTraceContext(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Contextにスパン情報がない場合、空のTraceContextを返す", func(t *testing.T) {
			t.Parallel()
			expected := &TraceContext{}

			actual := ExtractTraceContext(context.Background())
			assert.Equal(t, expected, actual)
		})

		t.Run("Contextにスパン情報がある場合、正しいTraceContextを返す", func(t *testing.T) {
			t.Parallel()

			ctx, end := NewStubSpanContext(t)
			defer end()

			span := trace.SpanFromContext(ctx).SpanContext()
			assert.True(t, span.IsValid())

			expected := &TraceContext{
				traceID: span.TraceID().String(),
				spanID:  span.SpanID().String(),
			}
			actual := ExtractTraceContext(ctx)
			assert.Equal(t, expected, actual)
		})
	})
}

func TestStartSpanWithParent(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("親spanのtrace/span IDを引き継いだ子spanを開始する", func(t *testing.T) {
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
		})
	})
}

func TestTraceContext_IDs(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("TraceID/SpanIDのgetterはコンストラクタに渡した値を返す", func(t *testing.T) {
			t.Parallel()

			traceID := "trace-id-123"
			spanID := "span-id-456"
			tc := TraceContext{
				traceID: traceID,
				spanID:  spanID,
			}

			assert.Equal(t, traceID, tc.TraceID())
			assert.Equal(t, spanID, tc.SpanID())
		})
	})
}
