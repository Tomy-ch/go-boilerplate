package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func Test_mustOpenSink(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("解決可能なsinkはWriteSyncerを返す", func(t *testing.T) {
			t.Parallel()
			assert.NotNil(t, mustOpenSink("stdout"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未登録スキームのsinkはpanicする", func(t *testing.T) {
			t.Parallel()
			assert.Panics(t, func() { _ = mustOpenSink("bogus-scheme://x") })
		})
	})
}

func Test_buildLogger(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定レベル以上のログを出力先へ書き込む", func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			enc := zapcore.NewJSONEncoder(encoderConfig(zapcore.LowercaseLevelEncoder))
			l := buildLogger(enc, zapcore.AddSync(&buf), zapcore.InfoLevel, zapcore.ErrorLevel, true, nil)

			l.Info(context.Background(), "hello")
			assert.Contains(t, buf.String(), "hello")
		})

		t.Run("指定レベル未満のログは書き込まれない", func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			enc := zapcore.NewJSONEncoder(encoderConfig(zapcore.LowercaseLevelEncoder))
			l := buildLogger(enc, zapcore.AddSync(&buf), zapcore.InfoLevel, zapcore.ErrorLevel, true, nil)

			l.Debug(context.Background(), "should not appear")
			assert.Empty(t, buf.String())
		})

		t.Run("json設定でErrorログのstacktraceが配列で出力される", func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			enc := zapcore.NewJSONEncoder(encoderConfig(zapcore.LowercaseLevelEncoder))
			l := buildLogger(enc, zapcore.AddSync(&buf), zapcore.InfoLevel, zapcore.ErrorLevel, true, nil)
			l.Error(context.Background(), "simulated server error")

			var got map[string]any
			require.NoError(t, json.Unmarshal(bytes.TrimRight(buf.Bytes(), "\n"), &got))

			st, ok := got["stacktrace"]
			require.True(t, ok, "stacktrace key must exist")
			arr, ok := st.([]any)
			require.True(t, ok, "stacktrace must be JSON array, got %T", st)
			require.NotEmpty(t, arr)
		})

		t.Run("console設定ではstacktraceが改行付きの単一文字列として出力される", func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			enc := zapcore.NewConsoleEncoder(encoderConfig(zapcore.CapitalColorLevelEncoder))
			l := buildLogger(enc, zapcore.AddSync(&buf), zapcore.DebugLevel, zapcore.ErrorLevel, false, nil)
			l.Error(context.Background(), "simulated server error")

			out := buf.String()
			require.Contains(t, out, "\n", "console output must keep newlines")
			require.NotContains(t, out, `"stacktrace":[`, "console output must not contain JSON array form of stack")
			assert.GreaterOrEqual(t, strings.Count(out, "\n"), 2)
		})
	})
}

func TestNewJSONLogger(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JSON用Loggerを返す", func(t *testing.T) {
			t.Parallel()
			logger := NewJSONLogger(LevelInfo(), LevelError(), nil)
			require.NotNil(t, logger)
		})
	})
}

func TestNewConsoleLogger(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("console用Loggerを返す", func(t *testing.T) {
			t.Parallel()
			logger := NewConsoleLogger(LevelDebug(), LevelWarn(), nil)
			require.NotNil(t, logger)
		})
	})
}
