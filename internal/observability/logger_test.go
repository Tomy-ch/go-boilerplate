package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLayerTracer_logFields(t *testing.T) {
	t.Run("スパンが無い場合は trace_id/span_id は含まれない", func(t *testing.T) {
		t.Parallel()

		lt := LayerTracer{layer: "controller", pkgName: "pkg", funcName: "fn"}
		ctx := context.Background()

		fields := lt.logFields(ctx, lt.layer, lt.pkgName, lt.funcName, "ev", "name", zap.String("extra", "v1"))

		// ロガーに fields を付与して JSON 出力を取得
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
		logger.With(fields...).Info("test")

		var out map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &out))

		require.Equal(t, "ev", out["span_event"])
		require.Equal(t, "name", out["span_name"])
		require.Equal(t, "controller", out["layer"])
		require.Equal(t, "pkg", out["package"])
		require.Equal(t, "fn", out["func"])
		require.Equal(t, "v1", out["extra"])

		// スパン ID 関連は存在しない
		_, hasTrace := out["trace_id"]
		_, hasSpan := out["span_id"]
		require.False(t, hasTrace)
		require.False(t, hasSpan)
	})

	t.Run("スパンがある場合は trace_id/span_id を含む", func(t *testing.T) {
		t.Parallel()

		// SDK の TracerProvider を用意して実際にスパンを作る
		tp := sdktrace.NewTracerProvider()
		// set global to ensure otel.Tracer works (not strictly necessary)
		otel.SetTracerProvider(tp)
		defer func() { _ = tp.Shutdown(context.Background()) }()

		tracer := tp.Tracer("test")
		ctx, sp := tracer.Start(context.Background(), "s")
		defer sp.End()

		lt := LayerTracer{layer: "usecase", pkgName: "mypkg", funcName: "Do"}
		fields := lt.logFields(ctx, lt.layer, lt.pkgName, lt.funcName, "ev2", "name2")

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
		logger.With(fields...).Info("test")

		var out map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &out))

		require.Equal(t, "ev2", out["span_event"])
		require.Equal(t, "name2", out["span_name"])
		require.Equal(t, "usecase", out["layer"])
		require.Equal(t, "mypkg", out["package"])
		require.Equal(t, "Do", out["func"])

		// trace_id / span_id が存在し、空でないこと
		tid, ok := out["trace_id"].(string)
		require.True(t, ok)
		require.NotEmpty(t, tid)
		sid, ok := out["span_id"].(string)
		require.True(t, ok)
		require.NotEmpty(t, sid)
	})
}
