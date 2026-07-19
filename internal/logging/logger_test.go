package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// newExtractSpyLogger は、extract の呼び出しを記録する logger と呼び出し有無のフラグを返すテストヘルパーです。
func newExtractSpyLogger(t *testing.T) (*logger, *bool) {
	t.Helper()

	called := false
	return &logger{
		log: zap.NewNop(),
		extract: func(context.Context) (string, string, bool) {
			called = true
			return "t", "s", true
		},
	}, &called
}

func TestWithCore(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("coreがnilなら元のLoggerをそのまま返す", func(t *testing.T) {
			t.Parallel()
			base := NewConsoleLogger(LevelDebug(), LevelError(), nil)
			got := WithCore(base, nil)
			assert.Same(t, base, got)
		})

		t.Run("coreを渡すと別のLoggerを返し追加coreへTeeされる", func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{MessageKey: "msg"})
			extra := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.DebugLevel)

			base := NewConsoleLogger(LevelDebug(), LevelError(), nil)
			got := WithCore(base, extra)

			assert.NotSame(t, base, got)
			got.Info(context.Background(), "tee-test")
			assert.Contains(t, buf.String(), "tee-test")
		})

		t.Run("extractがTee後のLoggerへ伝播する", func(t *testing.T) {
			t.Parallel()

			base, called := newExtractSpyLogger(t)
			enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{MessageKey: "msg"})
			extra := zapcore.NewCore(enc, zapcore.AddSync(&bytes.Buffer{}), zapcore.DebugLevel)

			got, ok := WithCore(base, extra).(*logger)
			require.True(t, ok)
			require.NotNil(t, got.extract)
			got.extract(context.Background())
			assert.True(t, *called)
		})

		t.Run("追加coreは元Loggerの最小レベルでゲートされる", func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{MessageKey: "msg"})
			// 追加 core 自体は Debug まで通すが、元 Logger は Info のため Debug はゲートされる。
			extra := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.DebugLevel)

			base := NewConsoleLogger(LevelInfo(), LevelError(), nil)
			got := WithCore(base, extra)

			got.Debug(context.Background(), "gated-out")
			assert.NotContains(t, buf.String(), "gated-out")

			got.Info(context.Background(), "passed")
			assert.Contains(t, buf.String(), "passed")
		})

		t.Run("追加core自身のレベル未満のログはゲートされ書き込まれない", func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{MessageKey: "msg"})
			// 元 Logger は Debug のため tee は Info でも呼ばれるが、追加 core は Error 以上のみ有効。
			extra := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.ErrorLevel)

			base := NewConsoleLogger(LevelDebug(), LevelError(), nil)
			got := WithCore(base, extra)

			got.Info(context.Background(), "gated-info")
			assert.NotContains(t, buf.String(), "gated-info")

			got.Error(context.Background(), "passed-error")
			assert.Contains(t, buf.String(), "passed-error")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("logger以外のLogger実装はゲートできず受け取った値をそのまま返す", func(t *testing.T) {
			t.Parallel()

			// *logger 以外（ここでは nil interface）は core を内包しないため、
			// ゲートせず受け取った Logger をそのまま返す。
			var other Logger
			enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{MessageKey: "msg"})
			extra := zapcore.NewCore(enc, zapcore.AddSync(&bytes.Buffer{}), zapcore.DebugLevel)

			got := WithCore(other, extra)
			assert.Nil(t, got)
		})
	})
}

func Test_levelGatedCore_Check(t *testing.T) {
	t.Parallel()

	newGatedCore := func(t *testing.T) (levelGatedCore, *bytes.Buffer) {
		t.Helper()
		var buf bytes.Buffer
		enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{MessageKey: "msg"})
		inner := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.DebugLevel)
		return levelGatedCore{Core: inner, min: zapcore.WarnLevel}, &buf
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("min以上のレベルは自身をCheckedEntryへ追加し書き込みへ到達する", func(t *testing.T) {
			t.Parallel()

			gated, buf := newGatedCore(t)
			ce := gated.Check(zapcore.Entry{Level: zapcore.ErrorLevel, Message: "hi"}, nil)

			require.NotNil(t, ce)
			ce.Write()
			assert.Contains(t, buf.String(), "hi")
		})

		t.Run("min未満のレベルは追加せず受け取ったCheckedEntryをそのまま返す", func(t *testing.T) {
			t.Parallel()

			gated, _ := newGatedCore(t)
			ce := gated.Check(zapcore.Entry{Level: zapcore.InfoLevel, Message: "hi"}, nil)

			assert.Nil(t, ce)
		})
	})
}

func Test_levelGatedCore_With(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("minを保持しつつ内側coreへフィールドを伝播した新coreを返す", func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{MessageKey: "msg"})
			inner := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.DebugLevel)
			gated := levelGatedCore{Core: inner, min: zapcore.WarnLevel}

			got := gated.With([]zapcore.Field{zap.String("svc", "demo")})

			gc, ok := got.(levelGatedCore)
			require.True(t, ok)
			assert.Equal(t, zapcore.WarnLevel, gc.min)
			assert.False(t, gc.Enabled(zapcore.InfoLevel))
			assert.True(t, gc.Enabled(zapcore.WarnLevel))

			require.NoError(t, gc.Write(zapcore.Entry{Level: zapcore.WarnLevel, Message: "hi"}, nil))
			var m map[string]any
			require.NoError(t, json.Unmarshal(bytes.TrimRight(buf.Bytes(), "\n"), &m))
			assert.Equal(t, "demo", m["svc"])
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

		t.Run("extractがCallerSkip後のLoggerへ伝播する", func(t *testing.T) {
			t.Parallel()

			base, called := newExtractSpyLogger(t)

			child, ok := base.CallerSkip(1).(*logger)
			require.True(t, ok)
			require.NotNil(t, child.extract)
			child.extract(context.Background())
			assert.True(t, *called)
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

		t.Run("extractが子Loggerへ伝播する", func(t *testing.T) {
			t.Parallel()

			base, called := newExtractSpyLogger(t)

			child, ok := base.Named("child").(*logger)
			require.True(t, ok)
			require.NotNil(t, child.extract)
			child.extract(context.Background())
			assert.True(t, *called)
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
			expectedError := xerrors.New("boom")
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
			l.Debug(context.Background(), "debug message", String("key", "value"))

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
			l.Info(context.Background(), "info message", String("key", "value"))

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
			l.Warn(context.Background(), "warn message", String("key", "value"))

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
			l.Error(context.Background(), "error message", String("key", "value"))

			out := buf.String()
			assert.Contains(t, out, "error message")
			assert.Contains(t, out, "key")
			assert.Contains(t, out, "value")
		})
	})
}

func Test_logger_injectTrace(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("extractorがtrueを返すとtrace_idとspan_idが正しいキーで先頭2要素に注入される", func(t *testing.T) {
			t.Parallel()

			l := &logger{
				log: zap.NewNop(),
				extract: func(context.Context) (string, string, bool) {
					return "trace-abc", "span-xyz", true
				},
			}

			got := l.injectTrace(context.Background(), []*Field{String("key", "value")})

			require.Len(t, got, 3)
			assert.Equal(t, String(TraceIDKey, "trace-abc"), got[0])
			assert.Equal(t, String(SpanIDKey, "span-xyz"), got[1])
			assert.Equal(t, String("key", "value"), got[2])
		})

		t.Run("extractorがnilならtraceは注入されずそのまま出力される", func(t *testing.T) {
			t.Parallel()

			l, buf := newBufLogger()
			l.Info(context.Background(), "no extractor", String("key", "value"))

			out := buf.String()
			assert.Contains(t, out, "no extractor")
			assert.Contains(t, out, "value")
			assert.NotContains(t, out, TraceIDKey)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("extractorがfalseを返すとtraceは注入されない", func(t *testing.T) {
			t.Parallel()

			l, buf := newBufLogger()
			l.extract = func(context.Context) (string, string, bool) {
				return "", "", false
			}
			l.Info(context.Background(), "gated trace", String("key", "value"))

			out := buf.String()
			assert.Contains(t, out, "gated trace")
			assert.Contains(t, out, "value")
			assert.NotContains(t, out, TraceIDKey)
		})
	})
}
