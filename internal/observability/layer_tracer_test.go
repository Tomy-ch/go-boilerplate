package observability

import (
	"context"
	"sync"
	"testing"

	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type recordingExporter struct {
	mu    sync.Mutex
	names []string
}

func (e *recordingExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, s := range spans {
		e.names = append(e.names, s.Name())
	}
	return nil
}

func (*recordingExporter) Shutdown(context.Context) error { return nil }

func (e *recordingExporter) recorded() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.names...)
}

func TestLayerTracer_Start(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("funcNameが既に設定されている場合、layer.pkg.func形式のspan名で開始する", func(t *testing.T) {
			t.Parallel()

			exp := &recordingExporter{}
			tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
			defer func() { _ = tp.Shutdown(context.Background()) }()

			lt := LayerTracer{
				tracer: tp.Tracer("test"),
				layer:  Usecase, pkgName: "pkg", funcName: "Fn",
			}
			ctx, end := lt.Start(context.Background())
			end()

			require.NotNil(t, ctx)
			assert.Equal(t, []string{"usecase.pkg.Fn"}, exp.recorded())
		})

		t.Run("funcNameが空の場合、getCallerFullNameによって設定される", func(t *testing.T) {
			t.Parallel()

			tracer, shutdown := newTestTracer(t)
			defer shutdown()

			lt := LayerTracer{
				tracer: tracer,
				layer:  "usecase", pkgName: "pkg", funcName: "",
			}
			ctx := context.Background()
			ctx, end := lt.Start(ctx)
			end()
			require.NotNil(t, ctx)
		})
	})
}

func TestLayerTracer_StartWithSuffix(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("funcNameが空の場合、getCallerFullNameによって設定される", func(t *testing.T) {
			t.Parallel()

			tracer, shutdown := newTestTracer(t)
			defer shutdown()

			lt := LayerTracer{
				tracer: tracer,
				layer:  "controller", pkgName: "p", funcName: "",
			}
			ctx, end := lt.StartWithSuffix(context.Background(), "DB")
			end()
			require.NotNil(t, ctx)
		})

		t.Run("optionalNameを指定した場合、span名にサフィックスが付与される", func(t *testing.T) {
			t.Parallel()

			exp := &recordingExporter{}
			tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
			defer func() { _ = tp.Shutdown(context.Background()) }()

			lt := LayerTracer{
				tracer: tp.Tracer("test"),
				layer:  Controller, pkgName: "p", funcName: "F",
			}
			ctx, end := lt.StartWithSuffix(context.Background(), "DB")
			end()

			require.NotNil(t, ctx)
			assert.Equal(t, []string{"controller.p.F.DB"}, exp.recorded())
		})

		t.Run("optionalNameが空の場合、サフィックス無しでspanを開始する", func(t *testing.T) {
			t.Parallel()

			tracer, shutdown := newTestTracer(t)
			defer shutdown()

			lt := LayerTracer{
				tracer: tracer,
				layer:  "controller", pkgName: "p", funcName: "F",
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

		t.Run("成功時はspan内で関数を実行し値を返す", func(t *testing.T) {
			t.Parallel()

			tracer, shutdown := newTestTracer(t)
			defer shutdown()

			lt := LayerTracer{tracer: tracer}

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

			lt := LayerTracer{tracer: tracer}

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

func TestLayerTracer_makeSpanName(t *testing.T) {
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

func newTestTracer(t *testing.T) (trace.Tracer, func()) {
	t.Helper()
	tp := sdktrace.NewTracerProvider()
	tracer := tp.Tracer("test")
	return tracer, func() { _ = tp.Shutdown(context.Background()) }
}

func TestLayerTracer_startSpan(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("optionalNameを付与したspan名で子spanを開始し終了関数を返す", func(t *testing.T) {
			t.Parallel()

			exp := &recordingExporter{}
			tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
			defer func() { _ = tp.Shutdown(context.Background()) }()

			lt := LayerTracer{
				tracer: tp.Tracer("test"),
				layer:  Usecase, pkgName: "pkg", funcName: "func",
			}

			spanCtx, end := lt.startSpan(context.Background(), "optional")

			assert.Empty(t, exp.recorded())
			end()

			assert.True(t, trace.SpanFromContext(spanCtx).SpanContext().IsValid())
			assert.Equal(t, []string{"usecase.pkg.func.optional"}, exp.recorded())
		})

		t.Run("親spanのある場合は同一traceの子spanになる", func(t *testing.T) {
			t.Parallel()

			exp := &recordingExporter{}
			tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
			defer func() { _ = tp.Shutdown(context.Background()) }()

			lt := LayerTracer{
				tracer: tp.Tracer("test"),
				layer:  Infra, pkgName: "pkg", funcName: "func",
			}
			parentCtx, parentSpan := tp.Tracer("test").Start(context.Background(), "parent")
			defer parentSpan.End()

			spanCtx, end := lt.startSpan(parentCtx, "")
			end()

			childSC := trace.SpanFromContext(spanCtx).SpanContext()
			assert.Equal(t, parentSpan.SpanContext().TraceID(), childSC.TraceID())
			assert.NotEqual(t, parentSpan.SpanContext().SpanID(), childSC.SpanID())
			assert.Equal(t, []string{"infrastructure.pkg.func"}, exp.recorded())
		})
	})
}

func TestLayerTracer_StartWithLink(t *testing.T) {
	t.Parallel()

	// 起点となる trace の識別子。W3C の traceparent 形式（version-traceid-spanid-flags）。
	const originTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	const originCarrier = "00-" + originTraceID + "-00f067aa0ba902b7-01"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("carrier が指す trace への link を持つ span を開く", func(t *testing.T) {
			t.Parallel()

			tp, recorded := NewRecordingTracerProvider(t)
			lt := NewTracerFactory(tp).Controller()

			_, endSpan := lt.StartWithLink(context.Background(), map[string]string{"traceparent": originCarrier})
			endSpan()

			spans := recorded()
			require.Len(t, spans, 1)
			require.Len(t, spans[0].Links(), 1, "起点 trace への link が 1 本張られること")
			assert.Equal(t, originTraceID, spans[0].Links()[0].SpanContext.TraceID().String())
		})

		t.Run("link を張っても親は呼び出し側の trace のまま", func(t *testing.T) {
			t.Parallel()

			tp, recorded := NewRecordingTracerProvider(t)
			lt := NewTracerFactory(tp).Controller()

			parentCtx, endParent := lt.Start(context.Background())
			_, endChild := lt.StartWithLink(parentCtx, map[string]string{"traceparent": originCarrier})
			endChild()
			endParent()

			spans := recorded()
			require.Len(t, spans, 2)
			// 起点 trace の子にしてしまうと、接続が続く限り起点の trace が閉じない。
			assert.NotEqual(t, originTraceID, spans[0].SpanContext().TraceID().String())
			assert.Equal(t, spans[1].SpanContext().TraceID(), spans[0].SpanContext().TraceID())
		})

		t.Run("carrier が空なら link 無しの span を開く", func(t *testing.T) {
			t.Parallel()

			tp, recorded := NewRecordingTracerProvider(t)
			lt := NewTracerFactory(tp).Controller()

			_, endSpan := lt.StartWithLink(context.Background(), map[string]string{"traceparent": ""})
			endSpan()

			spans := recorded()
			require.Len(t, spans, 1)
			assert.Empty(t, spans[0].Links(), "伝搬が途切れた event は link 無しで済ませること")
		})
	})
}

func Test_linkFromCarrier(t *testing.T) {
	t.Parallel()

	const originCarrier = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("carrier の trace context を link に変える", func(t *testing.T) {
			t.Parallel()

			link, ok := linkFromCarrier(map[string]string{"traceparent": originCarrier})

			require.True(t, ok)
			assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", link.SpanContext.TraceID().String())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("読めない carrier では link を作らない", func(t *testing.T) {
			t.Parallel()

			// 空だけでなく「形が崩れている」も link 無しに落ちること。doc が両方を約束している。
			for _, carrier := range []map[string]string{
				nil,
				{"traceparent": ""},
				{"traceparent": "garbage"},
			} {
				_, ok := linkFromCarrier(carrier)
				assert.False(t, ok)
			}
		})
	})
}
