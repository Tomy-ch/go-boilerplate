package observability

import (
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
			require.NotNil(t, actual.tracer)
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
			require.NotNil(t, actual.tracer)
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
			require.NotNil(t, actual.tracer)
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
			require.NotNil(t, actual.tracer)
		})
	})
}

func TestNewNoopSpanContext(t *testing.T) {
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
