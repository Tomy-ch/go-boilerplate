package logging

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Test_logger_CallerSkip(t *testing.T) {
	t.Parallel()

	log := zap.NewNop()
	baseLogger := &logger{log: log}

	skip := 3
	expected := &logger{
		log: log.WithOptions(zap.AddCallerSkip(skip)),
	}
	actual := baseLogger.CallerSkip(skip)
	require.Equal(t, expected, actual)
}

func Test_logger_Named(t *testing.T) {
	t.Parallel()

	log := zap.NewNop()
	baseLogger := &logger{log: log}
	name := "testLogger"

	expected := &logger{
		log: log.Named(name),
	}
	actual := baseLogger.Named(name)
	require.Equal(t, expected, actual)
}

func Test_logger_ConvertFields(t *testing.T) {
	t.Parallel()

	log := zap.NewNop()
	l := &logger{log: log}

	expectedString := "value1"
	expectedStrings := []string{"one", "two", "three"}
	expectedInt := 42
	expectedInt64 := int64(100)
	expectedFloat64 := 3.14
	expectedBool := true
	expectedError := error(nil)
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
		zap.Error(expectedError),
		zap.Any("key7", expectedAny),
	}
	actual := l.ConvertFields(fields)
	require.Equal(t, expected, actual)
}

func Test_logger_Debug(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	enc := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	core := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.DebugLevel)
	zlog := zap.New(core)

	l := &logger{log: zlog}

	l.Debug("debug message", String("key", "value"))

	out := buf.String()
	require.Contains(t, out, "debug message")
	require.Contains(t, out, "key")
	require.Contains(t, out, "value")
}

func Test_logger_Info(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	enc := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	core := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.DebugLevel)
	zlog := zap.New(core)

	l := &logger{log: zlog}

	l.Info("info message", String("key", "value"))

	out := buf.String()
	require.Contains(t, out, "info message")
	require.Contains(t, out, "key")
	require.Contains(t, out, "value")
}

func Test_logger_Warn(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	enc := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	core := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.DebugLevel)
	zlog := zap.New(core)

	l := &logger{log: zlog}

	l.Warn("warn message", String("key", "value"))

	out := buf.String()
	require.Contains(t, out, "warn message")
	require.Contains(t, out, "key")
	require.Contains(t, out, "value")
}

func Test_logger_Error(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	enc := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	core := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.DebugLevel)
	zlog := zap.New(core)

	l := &logger{log: zlog}

	l.Error("error message", String("key", "value"))

	out := buf.String()
	require.Contains(t, out, "error message")
	require.Contains(t, out, "key")
	require.Contains(t, out, "value")
}
