package logging

import (
	"bytes"
	"encoding/json"
	"strings"
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

// NewJSONLogger 相当（JSON encoder + array 化ラッパ）の buildLogger を組み、
// JSON 出力の stacktrace キーが配列になるエンドツーエンド経路を検証する。
func Test_buildLogger_jsonStacktraceIsArray(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("json設定でErrorログのstacktraceが配列で出力される", func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			enc := zapcore.NewJSONEncoder(encoderConfig(zapcore.LowercaseLevelEncoder))
			l := buildLogger(enc, zapcore.AddSync(&buf), zapcore.InfoLevel, zapcore.ErrorLevel, true)
			l.Error("simulated server error")

			var got map[string]any
			require.NoError(t, json.Unmarshal(bytes.TrimRight(buf.Bytes(), "\n"), &got))

			st, ok := got["stacktrace"]
			require.True(t, ok, "stacktrace key must exist")
			arr, ok := st.([]any)
			require.True(t, ok, "stacktrace must be JSON array, got %T", st)
			require.NotEmpty(t, arr)
		})
	})
}

// console エンコーダで wrap が適用されると一行 JSON 化して可読性が破壊されるため、
// buildLogger は console 設定では wrap を適用せず、zap 標準の改行付きスタックを保つ。
func Test_buildLogger_consoleStacktraceStaysMultiline(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("console設定ではstacktraceが改行付きの単一文字列として出力される", func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			enc := zapcore.NewConsoleEncoder(encoderConfig(zapcore.CapitalColorLevelEncoder))
			l := buildLogger(enc, zapcore.AddSync(&buf), zapcore.DebugLevel, zapcore.ErrorLevel, false)
			l.Error("simulated server error")

			out := buf.String()
			// console エンコーダ標準の改行+インデント形式が保たれる（一行 JSON 配列化していない）。
			require.Contains(t, out, "\n", "console output must keep newlines")
			require.NotContains(t, out, `"stacktrace":[`, "console output must not contain JSON array form of stack")
			// 少なくとも複数行に渡るスタックが出力されていること。
			assert.GreaterOrEqual(t, strings.Count(out, "\n"), 2)
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
