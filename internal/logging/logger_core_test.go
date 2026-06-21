package logging

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestBuildLogger(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定レベル以上のログを出力先へ書き込む", func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			enc := zapcore.NewJSONEncoder(encoderConfig(zapcore.LowercaseLevelEncoder))
			l := buildLogger(enc, zapcore.AddSync(&buf), zapcore.InfoLevel, zapcore.ErrorLevel, true)

			l.Info("hello")
			assert.Contains(t, buf.String(), "hello")
		})

		t.Run("指定レベル未満のログは書き込まれない", func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			enc := zapcore.NewJSONEncoder(encoderConfig(zapcore.LowercaseLevelEncoder))
			l := buildLogger(enc, zapcore.AddSync(&buf), zapcore.InfoLevel, zapcore.ErrorLevel, true)

			l.Debug("should not appear")
			assert.Empty(t, buf.String())
		})
	})
}

func TestNewJSONLogger(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JSON用Loggerを返す", func(t *testing.T) {
			t.Parallel()
			logger := NewJSONLogger(LevelInfo, LevelError)
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
			logger := NewConsoleLogger(LevelDebug, LevelWarn)
			require.NotNil(t, logger)
		})
	})
}
