package logging

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// newJSONStacktraceLogger は、JSON エンコード + stacktraceArrayCore ラップ付きの zap ロガーを生成する。
func newJSONStacktraceLogger(t *testing.T, stacktraceLevel zapcore.Level) (*zap.Logger, *bytes.Buffer) {
	t.Helper()

	var buf bytes.Buffer
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.StacktraceKey = "stacktrace"
	enc := zapcore.NewJSONEncoder(encCfg)
	base := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.DebugLevel)

	wrapped := wrapStacktraceCore(base, encCfg.StacktraceKey)
	zl := zap.New(wrapped, zap.AddStacktrace(stacktraceLevel))
	return zl, &buf
}

func Test_stacktraceArrayCore_Write(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("stacktrace付きエントリはstacktraceが行配列としてJSON出力される", func(t *testing.T) {
			t.Parallel()

			zl, buf := newJSONStacktraceLogger(t, zapcore.ErrorLevel)
			zl.Error("boom")

			var got map[string]any
			require.NoError(t, json.Unmarshal(buf.Bytes(), &got))

			raw, ok := got["stacktrace"]
			require.True(t, ok, "stacktrace key must exist")

			arr, ok := raw.([]any)
			require.True(t, ok, "stacktrace must be JSON array, got %T", raw)
			require.NotEmpty(t, arr)
			for _, v := range arr {
				_, ok := v.(string)
				assert.True(t, ok, "each stacktrace element must be a string, got %T", v)
			}
		})

		t.Run("stacktraceLevel未満のエントリはstacktraceフィールドが付与されない", func(t *testing.T) {
			t.Parallel()

			zl, buf := newJSONStacktraceLogger(t, zapcore.ErrorLevel)
			zl.Info("just info")

			var got map[string]any
			require.NoError(t, json.Unmarshal(buf.Bytes(), &got))

			_, ok := got["stacktrace"]
			assert.False(t, ok, "stacktrace must not be present below configured level")
		})
	})
}

func Test_stacktraceArrayCore_With(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Withで付与したフィールドが後続出力に伝播する", func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			encCfg := zap.NewProductionEncoderConfig()
			encCfg.StacktraceKey = "stacktrace"
			enc := zapcore.NewJSONEncoder(encCfg)
			base := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.DebugLevel)

			wrapped := wrapStacktraceCore(base, "stacktrace").
				With([]zapcore.Field{zap.String("svc", "demo")})

			zl := zap.New(wrapped)
			zl.Info("hello")

			var got map[string]any
			require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
			assert.Equal(t, "demo", got["svc"])
		})
	})
}
