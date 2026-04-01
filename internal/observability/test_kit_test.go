package observability

import (
	"testing"

	"boilerplate-go/internal/logging"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewNoopTracerFactory(t *testing.T) {
	t.Parallel()
	lf := logging.NewTestLogFieldBuilder(t)

	tp := noop.NewTracerProvider()

	actual := NewNoopTracerFactory(t)
	require.Equal(t, lf, actual.(*tracerFactory).lf)
	require.NotNil(t, actual.(*tracerFactory).log)
	require.Equal(t, tp, actual.(*tracerFactory).tp)
}

func TestNewMockControllerLayerTracer(t *testing.T) {
	t.Parallel()

	actual := NewMockControllerLayerTracer(t)
	require.Equal(t, Controller, actual.layer)
	require.NotNil(t, actual.tracer)
	require.NotNil(t, actual.log)
	require.NotNil(t, actual.lf)
}

func TestNewMockUsecaseLayerTracer(t *testing.T) {
	t.Parallel()

	actual := NewMockUsecaseLayerTracer(t)
	require.Equal(t, Usecase, actual.layer)
	require.NotNil(t, actual.tracer)
	require.NotNil(t, actual.log)
	require.NotNil(t, actual.lf)
}

func TestNewMockInfraLayerTracer(t *testing.T) {
	t.Parallel()

	actual := NewMockInfraLayerTracer(t)
	require.Equal(t, Infra, actual.layer)
	require.NotNil(t, actual.tracer)
	require.NotNil(t, actual.log)
	require.NotNil(t, actual.lf)
}

func TestNewNoopLayerTracer(t *testing.T) {
	t.Parallel()

	actual := NewNoopLayerTracer(t)
	require.Equal(t, layer, actual.layer)
	require.Equal(t, pkg, actual.pkgName)
	require.Equal(t, logging.NewTestLogFieldBuilder(t), actual.lf)
	require.NotNil(t, actual.log)
}

func TestNewNoopSpanContext(t *testing.T) {
	ctx, span := NewStubSpanContext(t)
	require.NotEmpty(t, ctx)
	require.NotNil(t, span)

	spanCtx := trace.SpanFromContext(ctx)
	require.True(t, spanCtx.SpanContext().IsValid())
	defer span()
}
