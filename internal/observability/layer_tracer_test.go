package observability

import (
	"context"
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/logging"
	"boilerplate-go/pkg/xerrors"

	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestLayerTracer_fullName(t *testing.T) {
	t.Parallel()

	lt := LayerTracer{layer: "usecase", pkgName: "mypkg", funcName: "Do"}
	require.Equal(t, "usecase.mypkg.Do", lt.fullName(""))
	require.Equal(t, "usecase.mypkg.Do.Optional", lt.fullName("Optional"))
}

func TestLayerTracer(t *testing.T) {
	t.Parallel()

	t.Run("funcName が既に設定されている場合", func(t *testing.T) {
		t.Parallel()

		tracer, shutdown := newTestTracer(t)
		defer shutdown()

		logger := logging.NewTestInstance(t)
		lf := logging.NewLogFields(config.NewObservabilityConfig(config.MockConfigForTest(t)))

		lt := LayerTracer{
			log: logger, tracer: tracer, lf: lf,
			layer: "usecase", pkgName: "pkg", funcName: "Fn",
		}
		ctx := context.Background()
		ctx, end := lt.Start(ctx)
		end()
		require.NotNil(t, ctx)
	})

	t.Run("funcName が空の場合 getCallerFullName によって設定される", func(t *testing.T) {
		t.Parallel()

		tracer, shutdown := newTestTracer(t)
		defer shutdown()

		logger := logging.NewTestInstance(t)
		lf := logging.NewLogFields(config.NewObservabilityConfig(config.MockConfigForTest(t)))

		lt := LayerTracer{
			log: logger, tracer: tracer, lf: lf,
			layer: "usecase", pkgName: "pkg", funcName: "",
		}
		ctx := context.Background()
		ctx, end := lt.Start(ctx)
		end()
		require.NotNil(t, ctx)
	})

	t.Run("funcName が空の場合 getCallerFullName によって設定される", func(t *testing.T) {
		t.Parallel()

		tracer, shutdown := newTestTracer(t)
		defer shutdown()

		logger := logging.NewTestInstance(t)
		lf := logging.NewLogFields(config.NewObservabilityConfig(config.MockConfigForTest(t)))

		lt := LayerTracer{
			log: logger, tracer: tracer, lf: lf,
			layer: "controller", pkgName: "p", funcName: "",
		}
		ctx, end := lt.StartOptional(context.Background(), "DB")
		end()
		require.NotNil(t, ctx)
	})

	t.Run("optionalName を指定した場合", func(t *testing.T) {
		t.Parallel()

		tracer, shutdown := newTestTracer(t)
		defer shutdown()

		logger := logging.NewTestInstance(t)
		lf := logging.NewLogFields(config.NewObservabilityConfig(config.MockConfigForTest(t)))

		lt := LayerTracer{
			log: logger, tracer: tracer, lf: lf,
			layer: "controller", pkgName: "p", funcName: "F",
		}
		ctx, end := lt.StartOptional(context.Background(), "DB")
		end()
		require.NotNil(t, ctx)
	})

	t.Run("optionalName が空の場合", func(t *testing.T) {
		t.Parallel()

		tracer, shutdown := newTestTracer(t)
		defer shutdown()

		logger := logging.NewTestInstance(t)
		lf := logging.NewLogFields(config.NewObservabilityConfig(config.MockConfigForTest(t)))

		lt := LayerTracer{
			log: logger, tracer: tracer, lf: lf,
			layer: "controller", pkgName: "p", funcName: "F",
		}
		ctx, end := lt.StartOptional(context.Background(), "")
		end()
		require.NotNil(t, ctx)
	})
}

func TestWithDomainSpan(t *testing.T) {
	t.Parallel()

	t.Run("成功時は値を返しログにstart/endが出る", func(t *testing.T) {
		t.Parallel()

		tracer, shutdown := newTestTracer(t)
		defer shutdown()

		logger := logging.NewTestInstance(t)
		lf := logging.NewLogFields(config.NewObservabilityConfig(config.MockConfigForTest(t)))

		lt := LayerTracer{
			log: logger, tracer: tracer, lf: lf,
		}

		ctx, v, err := WithDomainSpan(
			context.Background(), lt, "pkg", "Func",
			func(_ context.Context) (string, error) {
				return "ok", nil
			})
		require.NoError(t, err)
		require.Equal(t, "ok", v)
		require.NotNil(t, ctx)
	})

	t.Run("エラー時はゼロ値とエラーを返しログはendを含む", func(t *testing.T) {
		t.Parallel()

		tracer, shutdown := newTestTracer(t)
		defer shutdown()

		logger := logging.NewTestInstance(t)
		lf := logging.NewLogFields(config.NewObservabilityConfig(config.MockConfigForTest(t)))

		lt := LayerTracer{
			log: logger, tracer: tracer, lf: lf,
		}

		ctx, v, err := WithDomainSpan(
			context.Background(), lt, "pkg", "Func",
			func(_ context.Context) (string, error) {
				return "", xerrors.New("failure")
			})

		require.Error(t, err)

		require.Empty(t, v)
		require.NotNil(t, ctx)
	})
}

func newTracerProvider() *sdktrace.TracerProvider {
	return sdktrace.NewTracerProvider()
}

func newTestTracer(t *testing.T) (trace.Tracer, func()) {
	t.Helper()
	tp := newTracerProvider()
	tracer := tp.Tracer("test")
	return tracer, func() { _ = tp.Shutdown(context.Background()) }
}
