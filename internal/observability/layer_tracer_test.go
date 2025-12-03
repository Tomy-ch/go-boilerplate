package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/logging"
	"boilerplate-go/pkg/xerrors"

	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLayerTracer_fullName(t *testing.T) {
	t.Parallel()

	lt := LayerTracer{layer: "usecase", pkgName: "mypkg", funcName: "Do"}
	require.Equal(t, "usecase.mypkg.Do", lt.fullName(""))
	require.Equal(t, "usecase.mypkg.Do.Optional", lt.fullName("Optional"))
}

func TestLayerTracer_Start(t *testing.T) {
	t.Parallel()

	t.Run("funcName が既に設定されている場合", func(t *testing.T) {
		t.Parallel()

		tracer, shutdown := newTestTracer(t)
		defer shutdown()

		logger, buf := newZapLoggerBuffer()
		lf := logging.NewLogFields(config.NewObservabilityConfig(config.MockConfigForTest(t)))

		lt := LayerTracer{
			log: logger, tracer: tracer, lf: lf,
			layer: "usecase", pkgName: "pkg", funcName: "Fn",
		}
		ctx := context.Background()
		ctx, end := lt.Start(ctx)
		end()
		require.NotNil(t, ctx)

		objs := decodeZapJSONBuffer(t, buf)
		require.GreaterOrEqual(t, len(objs), 2)

		require.Equal(t, SpanEventStart, objs[0]["span_event"])
		require.Equal(t, SpanEventEnd, objs[1]["span_event"])
		_, hasDur := objs[1]["latency_ms"]
		require.True(t, hasDur)
	})

	t.Run("funcName が空の場合 getCallerFullName によって設定される", func(t *testing.T) {
		t.Parallel()

		tracer, shutdown := newTestTracer(t)
		defer shutdown()

		logger, buf := newZapLoggerBuffer()
		lf := logging.NewLogFields(config.NewObservabilityConfig(config.MockConfigForTest(t)))

		lt := LayerTracer{
			log: logger, tracer: tracer, lf: lf,
			layer: "usecase", pkgName: "pkg", funcName: "",
		}
		ctx := context.Background()
		ctx, end := lt.Start(ctx)
		end()
		require.NotNil(t, ctx)

		objs := decodeZapJSONBuffer(t, buf)
		require.GreaterOrEqual(t, len(objs), 2)

		sn, ok := objs[0]["span_name"].(string)
		require.True(t, ok)
		require.NotEmpty(t, sn)
	})
}

func TestLayerTracer_StartOptional(t *testing.T) {
	t.Parallel()

	t.Run("funcName が空の場合 getCallerFullName によって設定される", func(t *testing.T) {
		t.Parallel()

		tracer, shutdown := newTestTracer(t)
		defer shutdown()

		logger, buf := newZapLoggerBuffer()
		lf := logging.NewLogFields(config.NewObservabilityConfig(config.MockConfigForTest(t)))

		lt := LayerTracer{
			log: logger, tracer: tracer, lf: lf,
			layer: "controller", pkgName: "p", funcName: "",
		}
		ctx, end := lt.StartOptional(context.Background(), "DB")
		end()
		require.NotNil(t, ctx)

		objs := decodeZapJSONBuffer(t, buf)
		require.GreaterOrEqual(t, len(objs), 2)
		require.Contains(t, objs[0]["span_name"], ".DB")
		require.Contains(t, objs[1]["span_name"], ".DB")
	})

	t.Run("optionalName を指定した場合", func(t *testing.T) {
		t.Parallel()

		tracer, shutdown := newTestTracer(t)
		defer shutdown()

		logger, buf := newZapLoggerBuffer()
		lf := logging.NewLogFields(config.NewObservabilityConfig(config.MockConfigForTest(t)))

		lt := LayerTracer{
			log: logger, tracer: tracer, lf: lf,
			layer: "controller", pkgName: "p", funcName: "F",
		}
		ctx, end := lt.StartOptional(context.Background(), "DB")
		end()
		require.NotNil(t, ctx)

		objs := decodeZapJSONBuffer(t, buf)
		require.GreaterOrEqual(t, len(objs), 2)
		require.Contains(t, objs[0]["span_name"], ".DB")
		require.Contains(t, objs[1]["span_name"], ".DB")
	})

	t.Run("optionalName が空の場合", func(t *testing.T) {
		t.Parallel()

		tracer, shutdown := newTestTracer(t)
		defer shutdown()

		logger, buf := newZapLoggerBuffer()
		lf := logging.NewLogFields(config.NewObservabilityConfig(config.MockConfigForTest(t)))

		lt := LayerTracer{
			log: logger, tracer: tracer, lf: lf,
			layer: "controller", pkgName: "p", funcName: "F",
		}
		ctx, end := lt.StartOptional(context.Background(), "")
		end()
		require.NotNil(t, ctx)

		objs := decodeZapJSONBuffer(t, buf)
		require.GreaterOrEqual(t, len(objs), 2)
		sn0 := objs[0]["span_name"].(string)
		sn1 := objs[1]["span_name"].(string)
		require.NotContains(t, sn0, ".DB")
		require.NotContains(t, sn1, ".DB")
	})
}

func newZapLoggerBuffer() (*zap.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	encCfg := zapcore.EncoderConfig{
		MessageKey:    "msg",
		LevelKey:      "level",
		TimeKey:       "ts",
		NameKey:       "logger",
		CallerKey:     "caller",
		StacktraceKey: "stack",
		EncodeLevel:   zapcore.LowercaseLevelEncoder,
		EncodeTime:    zapcore.ISO8601TimeEncoder,
		EncodeCaller:  zapcore.ShortCallerEncoder,
	}
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encCfg), zapcore.AddSync(&buf), zapcore.DebugLevel)
	logger := zap.New(core)
	return logger, &buf
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

func decodeZapJSONBuffer(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(buf.Bytes()))
	var objs []map[string]any
	for {
		var m map[string]any
		if err := dec.Decode(&m); err != nil {
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
		}
		objs = append(objs, m)
	}
	return objs
}

func TestWithDomainSpan_SuccessAndErrorPaths(t *testing.T) {
	t.Parallel()

	t.Run("成功時は値を返しログにstart/endが出る", func(t *testing.T) {
		t.Parallel()

		tracer, shutdown := newTestTracer(t)
		defer shutdown()

		logger, buf := newZapLoggerBuffer()
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

		objs := decodeZapJSONBuffer(t, buf)
		require.GreaterOrEqual(t, len(objs), 2)
		require.Equal(t, SpanEventStart, objs[0]["span_event"])
		require.Equal(t, SpanEventEnd, objs[1]["span_event"])
		require.Contains(t, objs[0]["span_name"], "domain.pkg.Func")
		_, has := objs[1]["latency_ms"]
		require.True(t, has)
	})

	t.Run("エラー時はゼロ値とエラーを返しログはendを含む", func(t *testing.T) {
		t.Parallel()

		tracer, shutdown := newTestTracer(t)
		defer shutdown()

		logger, buf := newZapLoggerBuffer()
		lf := logging.NewLogFields(config.NewObservabilityConfig(config.MockConfigForTest(t)))

		lt := LayerTracer{
			log: logger, tracer: tracer, lf: lf,
		}

		ctx, v, err := WithDomainSpan(
			context.Background(), lt, "pkg", "Func",
			func(_ context.Context) (string, error) {
				return "", xerrors.New("failure")
			})
		// we expect an error
		require.Error(t, err)
		// zero value for string
		require.Empty(t, v)
		require.NotNil(t, ctx)

		objs := decodeZapJSONBuffer(t, buf)
		require.GreaterOrEqual(t, len(objs), 2)
		require.Equal(t, SpanEventStart, objs[0]["span_event"])
		require.Equal(t, SpanEventEnd, objs[1]["span_event"])
		_, has := objs[1]["latency_ms"]
		require.True(t, has)
	})
}
