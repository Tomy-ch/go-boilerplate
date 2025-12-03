package observability

import (
	"context"
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/logging"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

func TestNewTracerFactory(t *testing.T) {
	t.Parallel()

	t.Run("TracerProvider と zap.Logger を渡すと TracerFactory を返す", func(t *testing.T) {
		t.Parallel()
		provided := otel.GetTracerProvider()
		lf := logging.NewLogFields(config.NewObservabilityConfig(config.MockConfigForTest(t)))
		log := zap.NewNop()

		expected := &tracerFactory{
			tp:  provided,
			log: log,
			lf:  lf,
		}
		actual := NewTracerFactory(provided, log, lf)
		require.Equal(t, expected, actual)
	})
}

func TestNewControllerTracer(t *testing.T) {
	t.Parallel()

	t.Run("TracerProvider を渡すと controller 用 LayerTracer を返す", func(t *testing.T) {
		t.Parallel()
		provided := otel.GetTracerProvider()
		lf := logging.NewLogFields(config.NewObservabilityConfig(config.MockConfigForTest(t)))
		log := zap.NewNop()
		actualTF := &tracerFactory{
			tp:  provided,
			log: log,
			lf:  lf,
		}

		actual := actualTF.Controller()
		require.NotNil(t, actual.tracer)
		require.Equal(t, tracerNameController, actual.layer)
		require.NotEmpty(t, actual.pkgName)

		ctx, end := actual.Start(context.Background())
		end()
		require.NotNil(t, ctx)
	})
}

func TestNewUsecaseTracer(t *testing.T) {
	t.Parallel()

	t.Run("TracerProvider を渡すと usecase 用 LayerTracer を返す", func(t *testing.T) {
		t.Parallel()
		provided := otel.GetTracerProvider()
		lf := logging.NewLogFields(config.NewObservabilityConfig(config.MockConfigForTest(t)))
		log := zap.NewNop()
		actualTF := &tracerFactory{
			tp:  provided,
			log: log,
			lf:  lf,
		}

		actual := actualTF.Usecase()
		require.NotNil(t, actual.tracer)
		require.Equal(t, tracerNameUsecase, actual.layer)
		require.NotEmpty(t, actual.pkgName)

		ctx, end := actual.Start(context.Background())
		end()
		require.NotNil(t, ctx)
	})
}

func TestNewInfrastructureTracer(t *testing.T) {
	t.Parallel()

	t.Run("TracerProvider を渡すと infrastructure 用 LayerTracer を返す", func(t *testing.T) {
		t.Parallel()
		provided := otel.GetTracerProvider()
		lf := logging.NewLogFields(config.NewObservabilityConfig(config.MockConfigForTest(t)))
		log := zap.NewNop()
		actualTF := &tracerFactory{
			tp:  provided,
			log: log,
			lf:  lf,
		}

		actual := actualTF.Infra()
		require.NotNil(t, actual.tracer)
		require.Equal(t, tracerNameInfrastructure, actual.layer)
		require.NotEmpty(t, actual.pkgName)

		ctx, end := actual.Start(context.Background())
		end()
		require.NotNil(t, ctx)
	})
}
