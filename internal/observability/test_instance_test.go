package observability

import (
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/logging"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewTestTracerFactory(t *testing.T) {
	t.Parallel()
	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewLogFields(obsCfg)

	tp := noop.NewTracerProvider()
	z := logging.NewTestInstance(t)

	expected := &tracerFactory{
		tp:  tp,
		log: z,
		lf:  lf,
	}
	actual := NewTestTracerFactory(t)
	require.Equal(t, expected, actual)
}

func TestNewTestSpanContext(t *testing.T) {
	ctx, span := NewTestSpanContext(t)
	require.NotEmpty(t, ctx)
	require.NotNil(t, span)

	spanCtx := trace.SpanFromContext(ctx)
	require.True(t, spanCtx.SpanContext().IsValid())
	defer span()
}
