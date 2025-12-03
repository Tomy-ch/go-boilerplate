package observability

import (
	"context"
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/logging"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

// NewTestTracerFactory は、テスト用のTracerFactoryを返します。
func NewTestTracerFactory(t *testing.T) TracerFactory {
	t.Helper()
	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewLogFields(obsCfg)

	tp := noop.NewTracerProvider()
	z := zap.NewNop()
	return NewTracerFactory(tp, z, lf)
}

// NewTestSpanContext は、テスト用のspan付きcontextを返します。
func NewTestSpanContext(t *testing.T) (context.Context, func()) {
	t.Helper()

	prev := otel.GetTracerProvider()

	tp := sdktrace.NewTracerProvider()
	tr := tp.Tracer("test-tracer")

	ctx, span := tr.Start(context.Background(), "test-span")

	return ctx, func() {
		span.End()
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(prev)
		require.Empty(t, otel.GetTracerProvider())
	}
}
