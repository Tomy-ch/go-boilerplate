package logging

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestWithCore(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("coreがnilなら元のLoggerをそのまま返す", func(t *testing.T) {
			t.Parallel()
			base := NewConsoleLogger(LevelDebug(), LevelError())
			got := WithCore(base, nil)
			assert.Same(t, base, got)
		})

		t.Run("coreを渡すと別のLoggerを返し追加coreへTeeされる", func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{MessageKey: "msg"})
			extra := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.DebugLevel)

			base := NewConsoleLogger(LevelDebug(), LevelError())
			got := WithCore(base, extra)

			assert.NotSame(t, base, got)
			got.Info("tee-test")
			assert.Contains(t, buf.String(), "tee-test")
		})

		t.Run("追加coreは元Loggerの最小レベルでゲートされる", func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{MessageKey: "msg"})
			// 追加 core 自体は Debug まで通すが、元 Logger は Info のため Debug はゲートされる。
			extra := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.DebugLevel)

			base := NewConsoleLogger(LevelInfo(), LevelError())
			got := WithCore(base, extra)

			got.Debug("gated-out")
			assert.NotContains(t, buf.String(), "gated-out")

			got.Info("passed")
			assert.Contains(t, buf.String(), "passed")
		})
	})
}

func Test_logger_CallerSkip(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("AddCallerSkipオプション適用済みのLoggerを返す", func(t *testing.T) {
			t.Parallel()

			log := zap.NewNop()
			baseLogger := &logger{log: log}

			skip := 3
			expected := &logger{
				log: log.WithOptions(zap.AddCallerSkip(skip)),
			}
			actual := baseLogger.CallerSkip(skip)
			assert.Equal(t, expected, actual)
		})
	})
}

func Test_logger_Named(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Named適用済みのLoggerを返す", func(t *testing.T) {
			t.Parallel()

			log := zap.NewNop()
			baseLogger := &logger{log: log}
			name := "testLogger"

			expected := &logger{
				log: log.Named(name),
			}
			actual := baseLogger.Named(name)
			assert.Equal(t, expected, actual)
		})
	})
}

func Test_logger_convertFields(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("各Fieldコンストラクタが対応するzap.Fieldへ変換される", func(t *testing.T) {
			t.Parallel()

			log := zap.NewNop()
			l := &logger{log: log}

			expectedString := "value1"
			expectedStrings := []string{"one", "two", "three"}
			expectedInt := 42
			expectedInt64 := int64(100)
			expectedFloat64 := 3.14
			expectedBool := true
			expectedError := errors.New("boom")
			expectedAny := "value7"

			fields := []*Field{
				String("key1", expectedString),
				Strings("key1s", expectedStrings),
				Int("key2", expectedInt),
				Int64("key3", expectedInt64),
				Float64("key4", expectedFloat64),
				Bool("key5", expectedBool),
				Error("key6", expectedError),
				Any("key7", expectedAny),
			}

			expected := []zap.Field{
				zap.String("key1", expectedString),
				zap.Strings("key1s", expectedStrings),
				zap.Int("key2", expectedInt),
				zap.Int64("key3", expectedInt64),
				zap.Float64("key4", expectedFloat64),
				zap.Bool("key5", expectedBool),
				zap.NamedError("key6", expectedError),
				zap.Any("key7", expectedAny),
			}
			actual := l.convertFields(fields)
			assert.Equal(t, expected, actual)
		})

		t.Run("未知のkindはdefault分岐でzap.Anyへ変換される", func(t *testing.T) {
			t.Parallel()

			log := zap.NewNop()
			l := &logger{log: log}

			fields := []*Field{{key: "unknown", kind: fieldUnknown}}

			expected := []zap.Field{
				zap.Any("unknown", nil),
			}
			actual := l.convertFields(fields)
			assert.Equal(t, expected, actual)
		})
	})
}

// newBufLogger は、書き込み内容を検証可能なバッファ付き logger を生成する。
func newBufLogger() (*logger, *bytes.Buffer) {
	var buf bytes.Buffer
	enc := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	core := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.DebugLevel)
	return &logger{log: zap.New(core)}, &buf
}

func Test_logger_Debug(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Debugレベルでメッセージとフィールドが出力される", func(t *testing.T) {
			t.Parallel()

			l, buf := newBufLogger()
			l.Debug("debug message", String("key", "value"))

			out := buf.String()
			assert.Contains(t, out, "debug message")
			assert.Contains(t, out, "key")
			assert.Contains(t, out, "value")
		})
	})
}

func Test_logger_Info(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Infoレベルでメッセージとフィールドが出力される", func(t *testing.T) {
			t.Parallel()

			l, buf := newBufLogger()
			l.Info("info message", String("key", "value"))

			out := buf.String()
			assert.Contains(t, out, "info message")
			assert.Contains(t, out, "key")
			assert.Contains(t, out, "value")
		})
	})
}

func Test_logger_Warn(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Warnレベルでメッセージとフィールドが出力される", func(t *testing.T) {
			t.Parallel()

			l, buf := newBufLogger()
			l.Warn("warn message", String("key", "value"))

			out := buf.String()
			assert.Contains(t, out, "warn message")
			assert.Contains(t, out, "key")
			assert.Contains(t, out, "value")
		})
	})
}

func Test_logger_Error(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Errorレベルでメッセージとフィールドが出力される", func(t *testing.T) {
			t.Parallel()

			l, buf := newBufLogger()
			l.Error("error message", String("key", "value"))

			out := buf.String()
			assert.Contains(t, out, "error message")
			assert.Contains(t, out, "key")
			assert.Contains(t, out, "value")
		})
	})
}
