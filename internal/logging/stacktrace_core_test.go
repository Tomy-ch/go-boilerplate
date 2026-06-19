package logging

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// メモリ書き込み可能な zap.Sink を 1 度だけ登録する。
// OutputPaths に "mem://buildlogger" を指定すると、test 側で取り出したバッファに書き込まれる。
var (
	memSinkOnce sync.Once
	memSinks    = sync.Map{} // key: URL.Host, val: *memSink
)

type memSink struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (m *memSink) Write(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.buf.Write(p)
}
func (m *memSink) Sync() error  { return nil }
func (m *memSink) Close() error { return nil }
func (m *memSink) Bytes() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]byte(nil), m.buf.Bytes()...)
}

// registerMemSink は、test 用メモリ sink スキームをプロセス内で 1 度だけ登録する。
func registerMemSink(t *testing.T) {
	t.Helper()
	memSinkOnce.Do(func() {
		require.NoError(t, zap.RegisterSink("mem", func(u *url.URL) (zap.Sink, error) {
			s := &memSink{}
			memSinks.Store(u.Host, s)
			return s, nil
		}))
	})
}

// readMemSink は、登録済みメモリ sink に書き込まれた内容を取り出す。
func readMemSink(t *testing.T, name string) []byte {
	t.Helper()
	v, ok := memSinks.Load(name)
	require.True(t, ok, "mem sink %q not found", name)
	s, ok := v.(*memSink)
	require.True(t, ok, "unexpected type stored for mem sink %q", name)
	return s.Bytes()
}

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

// 本番ロガー相当の Encoding=json + 非空 StacktraceKey 設定で buildLogger を組み、
// JSON 出力の stacktrace キーが配列になるエンドツーエンド経路を検証する。
func Test_buildLogger_jsonStacktraceIsArray(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("json設定でErrorログのstacktraceが配列で出力される", func(t *testing.T) {
			t.Parallel()

			registerMemSink(t)
			const sinkName = "buildlogger-json"
			cfg := zap.Config{
				Level:       zap.NewAtomicLevelAt(zapcore.InfoLevel),
				Encoding:    "json",
				OutputPaths: []string{"mem://" + sinkName},
				EncoderConfig: zapcore.EncoderConfig{
					MessageKey:    "msg",
					LevelKey:      "level",
					StacktraceKey: "stacktrace",
					EncodeLevel:   zapcore.LowercaseLevelEncoder,
				},
			}
			l, err := buildLogger(cfg, zapcore.ErrorLevel)
			require.NoError(t, err)
			l.Error("simulated server error")

			raw := readMemSink(t, sinkName)

			var got map[string]any
			require.NoError(t, json.Unmarshal(bytes.TrimRight(raw, "\n"), &got))

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

			registerMemSink(t)
			const sinkName = "buildlogger-console"
			cfg := zap.Config{
				Level:       zap.NewAtomicLevelAt(zapcore.DebugLevel),
				Encoding:    "console",
				OutputPaths: []string{"mem://" + sinkName},
				EncoderConfig: zapcore.EncoderConfig{
					MessageKey:    "msg",
					LevelKey:      "level",
					StacktraceKey: "Stack",
					EncodeLevel:   zapcore.CapitalLevelEncoder,
				},
			}
			l, err := buildLogger(cfg, zapcore.ErrorLevel)
			require.NoError(t, err)
			l.Error("simulated server error")

			raw := readMemSink(t, sinkName)
			out := string(raw)
			// console エンコーダ標準の改行+インデント形式が保たれる（一行 JSON 配列化していない）。
			require.Contains(t, out, "\n", "console output must keep newlines")
			require.NotContains(t, out, `"Stack":[`, "console output must not contain JSON array form of stack")
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
