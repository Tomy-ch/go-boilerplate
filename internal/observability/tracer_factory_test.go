package observability

import (
	"context"
	"testing"

	"go-boilerplate/internal/logging"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

func TestNewTracerFactory(t *testing.T) {
	t.Run("TracerProvider と zap.Logger を渡すと TracerFactory を返す", func(t *testing.T) {
		provided := otel.GetTracerProvider()
		lf := logging.NewTestLogFieldBuilder(t)
		log := logging.NewTestLogger(t)

		expected := &tracerFactory{
			tp:  provided,
			log: log,
			lf:  lf,
		}
		actual := NewTracerFactory(provided, log, lf)
		require.Equal(t, expected, actual)
	})
}

func Test_tracerFactory_Controller(t *testing.T) {
	t.Run("TracerProvider を渡すと controller 用 LayerTracer を返す", func(t *testing.T) {
		provided := otel.GetTracerProvider()
		lf := logging.NewTestLogFieldBuilder(t)
		log := logging.NewTestLogger(t)
		actualTF := &tracerFactory{
			tp:  provided,
			log: log,
			lf:  lf,
		}

		actual := actualTF.Controller()
		require.NotNil(t, actual)

		ctx, end := actual.Start(context.Background())
		end()
		require.NotNil(t, ctx)
	})
}

func Test_tracerFactory_Usecase(t *testing.T) {
	t.Run("TracerProvider を渡すと usecase 用 LayerTracer を返す", func(t *testing.T) {
		provided := otel.GetTracerProvider()
		lf := logging.NewTestLogFieldBuilder(t)
		log := logging.NewTestLogger(t)
		actualTF := &tracerFactory{
			tp:  provided,
			log: log,
			lf:  lf,
		}

		actual := actualTF.Usecase()
		require.NotNil(t, actual)

		ctx, end := actual.Start(context.Background())
		end()
		require.NotNil(t, ctx)
	})
}

func Test_tracerFactory_Infra(t *testing.T) {
	t.Run("TracerProvider を渡すと infrastructure 用 LayerTracer を返す", func(t *testing.T) {
		provided := otel.GetTracerProvider()
		lf := logging.NewTestLogFieldBuilder(t)
		log := logging.NewTestLogger(t)
		actualTF := &tracerFactory{
			tp:  provided,
			log: log,
			lf:  lf,
		}

		actual := actualTF.Infra()
		require.NotNil(t, actual)

		ctx, end := actual.Start(context.Background())
		end()
		require.NotNil(t, ctx)
	})
}

func Test_tracerFactory_newLayerTracer(t *testing.T) {
	t.Run("layer と pkgName を渡すと LayerTracer を返す", func(t *testing.T) {
		provided := otel.GetTracerProvider()
		lf := logging.NewTestLogFieldBuilder(t)
		log := logging.NewTestLogger(t)
		actualTF := &tracerFactory{
			tp:  provided,
			log: log,
			lf:  lf,
		}

		layer := Usecase
		pkgName := "pkg"
		actual := actualTF.newLayerTracer(layer, pkgName)
		require.NotNil(t, actual)
		require.Equal(t, layer, actual.layer)
		require.Equal(t, pkgName, actual.pkgName)
	})
}
