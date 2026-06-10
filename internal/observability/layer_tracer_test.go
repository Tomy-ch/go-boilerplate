package observability

import (
	"context"
	"testing"

	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestLayerTracer_makeSpanName(t *testing.T) {
	t.Parallel()

	lt := LayerTracer{layer: "usecase", pkgName: "mypkg", funcName: "Do"}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("optionalName無しはlayer.pkg.funcのみを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "usecase.mypkg.Do", lt.makeSpanName(""))
		})

		t.Run("optionalNameありはlayer.pkg.func.optionalを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "usecase.mypkg.Do.Optional", lt.makeSpanName("Optional"))
		})
	})
}

func Test_LayerTracer_Start(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("funcNameが既に設定されている場合、その値でspanを開始する", func(t *testing.T) {
			t.Parallel()

			tracer, shutdown := newTestTracer(t)
			defer shutdown()

			logger := logging.NewTestLogger(t)
			lf := logging.NewTestLogFieldBuilder(t)

			lt := LayerTracer{
				log: logger, tracer: tracer, lf: lf,
				layer: "usecase", pkgName: "pkg", funcName: "Fn",
			}
			ctx := context.Background()
			ctx, end := lt.Start(ctx)
			end()
			require.NotNil(t, ctx)
		})

		t.Run("funcNameが空の場合、getCallerFullNameによって設定される", func(t *testing.T) {
			t.Parallel()

			tracer, shutdown := newTestTracer(t)
			defer shutdown()

			logger := logging.NewTestLogger(t)
			lf := logging.NewTestLogFieldBuilder(t)

			lt := LayerTracer{
				log: logger, tracer: tracer, lf: lf,
				layer: "usecase", pkgName: "pkg", funcName: "",
			}
			ctx := context.Background()
			ctx, end := lt.Start(ctx)
			end()
			require.NotNil(t, ctx)
		})
	})
}

func Test_LayerTracer_StartWithSuffix(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("funcNameが空の場合、getCallerFullNameによって設定される", func(t *testing.T) {
			t.Parallel()

			tracer, shutdown := newTestTracer(t)
			defer shutdown()

			logger := logging.NewTestLogger(t)
			lf := logging.NewTestLogFieldBuilder(t)

			lt := LayerTracer{
				log: logger, tracer: tracer, lf: lf,
				layer: "controller", pkgName: "p", funcName: "",
			}
			ctx, end := lt.StartWithSuffix(context.Background(), "DB")
			end()
			require.NotNil(t, ctx)
		})

		t.Run("optionalNameを指定した場合、span名にサフィックスが付与される", func(t *testing.T) {
			t.Parallel()

			tracer, shutdown := newTestTracer(t)
			defer shutdown()

			logger := logging.NewTestLogger(t)
			lf := logging.NewTestLogFieldBuilder(t)

			lt := LayerTracer{
				log: logger, tracer: tracer, lf: lf,
				layer: "controller", pkgName: "p", funcName: "F",
			}
			ctx, end := lt.StartWithSuffix(context.Background(), "DB")
			end()
			require.NotNil(t, ctx)
		})

		t.Run("optionalNameが空の場合、サフィックス無しでspanを開始する", func(t *testing.T) {
			t.Parallel()

			tracer, shutdown := newTestTracer(t)
			defer shutdown()

			logger := logging.NewTestLogger(t)
			lf := logging.NewTestLogFieldBuilder(t)

			lt := LayerTracer{
				log: logger, tracer: tracer, lf: lf,
				layer: "controller", pkgName: "p", funcName: "F",
			}
			ctx, end := lt.StartWithSuffix(context.Background(), "")
			end()
			require.NotNil(t, ctx)
		})
	})
}

func TestRunWithSpan(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("成功時は値を返しログにstart/endが出る", func(t *testing.T) {
			t.Parallel()

			tracer, shutdown := newTestTracer(t)
			defer shutdown()

			logger := logging.NewTestLogger(t)
			lf := logging.NewTestLogFieldBuilder(t)

			lt := LayerTracer{
				log: logger, tracer: tracer, lf: lf,
			}

			ctx, v, err := RunWithSpan(
				context.Background(), lt, Usecase, "pkg", "Func",
				func(_ context.Context) (string, error) {
					return "ok", nil
				})
			require.NoError(t, err)
			assert.Equal(t, "ok", v)
			require.NotNil(t, ctx)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("コールバックがエラーを返した場合、ゼロ値とエラーを返す", func(t *testing.T) {
			t.Parallel()

			tracer, shutdown := newTestTracer(t)
			defer shutdown()

			logger := logging.NewTestLogger(t)
			lf := logging.NewTestLogFieldBuilder(t)

			lt := LayerTracer{
				log: logger, tracer: tracer, lf: lf,
			}

			ctx, v, err := RunWithSpan(
				context.Background(), lt, Usecase, "pkg", "Func",
				func(_ context.Context) (string, error) {
					return "", xerrors.New("failure")
				})

			require.Error(t, err)
			assert.Empty(t, v)
			require.NotNil(t, ctx)
		})
	})
}

func Test_makeSpanName(t *testing.T) {
	t.Parallel()

	expectedLayer := "layer"
	expectedPkgName := "pkg"
	expectedFuncName := "func"
	optionalName := "extra"
	expected := expectedLayer + delimiter + expectedPkgName + delimiter + expectedFuncName

	lt := LayerTracer{layer: layerName(expectedLayer), pkgName: expectedPkgName, funcName: expectedFuncName}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("optionalNameが空の場合、layer.pkg.funcのみを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, expected, lt.makeSpanName(""))
		})

		t.Run("optionalNameが指定された場合、layer.pkg.func.optionalを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, expected+delimiter+optionalName, lt.makeSpanName(optionalName))
		})
	})
}

func Test_startSpan(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("LayerTracerの設定でspanContextと終了関数を返す", func(t *testing.T) {
			t.Parallel()

			tracer, shutdown := newTestTracer(t)
			defer shutdown()

			logger := logging.NewTestLogger(t)
			lf := logging.NewTestLogFieldBuilder(t)

			lt := LayerTracer{
				log: logger, tracer: tracer, lf: lf,
				layer: Usecase, pkgName: "pkg", funcName: "func",
			}

			spanCtx, end := lt.startSpan(context.Background(), "optional")
			end()
			require.NotNil(t, spanCtx)
		})
	})
}

func newTestTracer(t *testing.T) (trace.Tracer, func()) {
	t.Helper()
	tp := sdktrace.NewTracerProvider()
	tracer := tp.Tracer("test")
	return tracer, func() { _ = tp.Shutdown(context.Background()) }
}
