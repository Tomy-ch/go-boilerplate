package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewTracerFactory(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("TracerProviderを渡すとTracerFactoryを返す", func(t *testing.T) {
			t.Parallel()

			provided := otel.GetTracerProvider()

			expected := &tracerFactory{
				tp: provided,
			}
			actual := NewTracerFactory(provided)
			assert.Equal(t, expected, actual)
		})
	})
}

func TestNewDisabledTracerFactory(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("スパンを送出しないTracerFactoryを返す", func(t *testing.T) {
			t.Parallel()

			expected := &tracerFactory{
				tp: noop.NewTracerProvider(),
			}
			actual := NewDisabledTracerFactory()

			assert.Equal(t, expected, actual)
		})
	})
}

func Test_tracerFactory_Controller(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Controllerレイヤ用LayerTracerを返す", func(t *testing.T) {
			t.Parallel()

			actualTF := &tracerFactory{tp: otel.GetTracerProvider()}

			actual := actualTF.Controller()
			require.NotNil(t, actual)

			ctx, end := actual.Start(context.Background())
			end()
			require.NotNil(t, ctx)
		})
	})
}

func Test_tracerFactory_Usecase(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Usecaseレイヤ用LayerTracerを返す", func(t *testing.T) {
			t.Parallel()

			actualTF := &tracerFactory{tp: otel.GetTracerProvider()}

			actual := actualTF.Usecase()
			require.NotNil(t, actual)

			ctx, end := actual.Start(context.Background())
			end()
			require.NotNil(t, ctx)
		})
	})
}

func Test_tracerFactory_Infra(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Infrastructureレイヤ用LayerTracerを返す", func(t *testing.T) {
			t.Parallel()

			actualTF := &tracerFactory{tp: otel.GetTracerProvider()}

			actual := actualTF.Infra()
			require.NotNil(t, actual)

			ctx, end := actual.Start(context.Background())
			end()
			require.NotNil(t, ctx)
		})
	})
}

func Test_tracerFactory_newLayerTracer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("layerとpkgNameを渡すとLayerTracerを返す", func(t *testing.T) {
			t.Parallel()

			actualTF := &tracerFactory{tp: otel.GetTracerProvider()}

			layer := Usecase
			pkgName := "pkg"
			actual := actualTF.newLayerTracer(layer, pkgName)
			require.NotNil(t, actual)
			assert.Equal(t, layer, actual.layer)
			assert.Equal(t, pkgName, actual.pkgName)
		})
	})
}
