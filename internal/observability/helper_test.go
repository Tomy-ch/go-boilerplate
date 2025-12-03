package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"boilerplate-go/internal/config"
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
		require.True(t, got)
	})

	t.Run("オブザーバビリティ有効だがスパン無し -> false", func(t *testing.T) {
		t.Parallel()

		cfg := config.MockConfigForTest(t)
		obsCfg := config.NewObservabilityConfig(cfg)

		ctx := context.Background()
		got := ShouldLogWithSpan(ctx, obsCfg)
		require.False(t, got)
	})

	t.Run("オブザーバビリティ無効だがスパンあり -> false", func(t *testing.T) {
		t.Parallel()

		obsCfg := &config.ObservabilityConfig{}

		tp := sdktrace.NewTracerProvider()
		tracer := tp.Tracer("test")
		ctx, sp := tracer.Start(context.Background(), "s")
		defer func() { sp.End(); _ = tp.Shutdown(context.Background()) }()

		got := ShouldLogWithSpan(ctx, obsCfg)
		require.False(t, got)
	})

	t.Run("オブザーバビリティ無効かつスパン無し -> false", func(t *testing.T) {
		t.Parallel()

		obsCfg := &config.ObservabilityConfig{}
		ctx := context.Background()
		got := ShouldLogWithSpan(ctx, obsCfg)
		require.False(t, got)
	})
}

func TestBuildSpanName(t *testing.T) {
	t.Parallel()

	layer := "layer"
	pkgName := "mypkg"
	funcName := "MyFunc"

	expected := "layer.mypkg.MyFunc"
	actual := BuildSpanName(layer, pkgName, funcName)
	require.Equal(t, expected, actual)
}

func TestExtractSpan(t *testing.T) {
	t.Run("Contextにスパン情報がない場合、空のTraceContextが返る", func(t *testing.T) {
		expected := TraceContext{}

		ctx := context.Background()
		actual := ExtractSpan(ctx)
		require.Equal(t, expected, actual)
	})

	t.Run("Contextにスパン情報がある場合、正しいTraceContextが返る", func(t *testing.T) {
		ctx, end := NewTestSpanContext(t)
		defer end()

		span := trace.SpanFromContext(ctx).SpanContext()
		require.True(t, span.IsValid())

		expected := TraceContext{
			traceID: span.TraceID().String(),
			spanID:  span.SpanID().String(),
		}
		actual := ExtractSpan(ctx)
		require.Equal(t, expected, actual)
	})
}

func TestTraceContext_IDs(t *testing.T) {
	t.Parallel()

	traceID := "trace-id-123"
	spanID := "span-id-456"
	tc := TraceContext{
		traceID: traceID,
		spanID:  spanID,
	}

	require.Equal(t, traceID, tc.TraceID())
	require.Equal(t, spanID, tc.SpanID())
}
