package logging

import (
	"fmt"
	"testing"
	"time"

	"boilerplate-go/internal/config"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewLogFields(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)

	lf := NewLogFields(obsCfg)
	require.NotNil(t, lf)
}

func TestLogFields_BuildHTTPRequestFields(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := NewLogFields(obsCfg)

	expectedPP := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	expectedQP := map[string][]string{
		"key3": {"value3"},
		"key4": {"value4a", "value4b"},
	}

	exampleInput := HTTPRequestLogInput{
		Method:   "GET",
		Path:     "/api/v1/example",
		URI:      "/api/v1/example?key3=value3&key4=value4a&key4=value4b",
		RemoteIP: "192.168.1.1",

		Host:          "example.com",
		Scheme:        "https",
		Proto:         "HTTP/1.1",
		UserAgent:     "Go-http-client/1.1",
		ContentType:   "application/json",
		ContentLength: 123,

		PathParams:  expectedPP,
		QueryParams: expectedQP,
	}

	t.Run("trace_idとspan_idが存在しない場合、trace_idとspan_idのフィールドは存在しない", func(t *testing.T) {
		t.Parallel()

		expected := []zap.Field{
			zap.String(MethodKey, exampleInput.Method),
			zap.String(PathKey, exampleInput.Path),
			zap.String(URIKey, exampleInput.URI),
			zap.String(RemoteIPKey, exampleInput.RemoteIP),

			zap.String(HostKey, exampleInput.Host),
			zap.String(SchemeKey, exampleInput.Scheme),
			zap.String(ProtoKey, exampleInput.Proto),
			zap.String(UserAgentKey, exampleInput.UserAgent),
			zap.String(ContentTypeKey, exampleInput.ContentType),
			zap.Int64(ContentLengthKey, exampleInput.ContentLength),

			zap.Any(QueryParamsKey, exampleInput.QueryParams),
			zap.Any(PathParamsKey, exampleInput.PathParams),
		}

		actual := lf.BuildHTTPRequestFields(exampleInput)
		require.Equal(t, expected, actual)
	})

	t.Run("trace_idとspan_idが存在する場合、trace_idとspan_idのフィールドが追加される", func(t *testing.T) {
		t.Parallel()

		input := exampleInput
		input.TraceID = "trace-id-123"
		input.SpanID = "span-id-456"

		expected := []zap.Field{
			zap.String(MethodKey, input.Method),
			zap.String(PathKey, input.Path),
			zap.String(URIKey, input.URI),
			zap.String(RemoteIPKey, input.RemoteIP),

			zap.String(HostKey, input.Host),
			zap.String(SchemeKey, input.Scheme),
			zap.String(ProtoKey, input.Proto),
			zap.String(UserAgentKey, input.UserAgent),
			zap.String(ContentTypeKey, input.ContentType),
			zap.Int64(ContentLengthKey, input.ContentLength),

			zap.Any(QueryParamsKey, input.QueryParams),
			zap.Any(PathParamsKey, input.PathParams),

			zap.String(TraceIDKey, input.TraceID),
			zap.String(SpanIDKey, input.SpanID),
		}

		actual := lf.BuildHTTPRequestFields(input)
		require.Equal(t, expected, actual)
	})
}

func TestLogFields_BuildResponseFields(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := NewLogFields(obsCfg)

	exampleInput := HTTPResponseLogInput{
		Status:    200,
		Method:    "POST",
		Path:      "/api/v1/resource",
		URI:       "/api/v1/resource?id=789",
		Latency:   150 * 1e6, // 150ms
		RequestID: "req-789",
	}

	t.Run("trace_idとspan_idが存在しない場合、trace_idとspan_idのフィールドは存在しない", func(t *testing.T) {
		t.Parallel()

		latencyMs := float64(exampleInput.Latency) / float64(time.Millisecond)

		expected := []zap.Field{
			zap.Int(StatusKey, exampleInput.Status),
			zap.String(MethodKey, exampleInput.Method),
			zap.String(PathKey, exampleInput.Path),
			zap.String(URIKey, exampleInput.URI),
			zap.Float64(LatencyKey, latencyMs),
			zap.String(RequestIDKey, exampleInput.RequestID),
		}

		actual := lf.BuildHTTPResponseFields(exampleInput)
		require.Equal(t, expected, actual)
	})

	t.Run("trace_idとspan_idが存在する場合、trace_idとspan_idのフィールドが追加される", func(t *testing.T) {
		t.Parallel()

		input := exampleInput
		input.TraceID = "trace-id-123"
		input.SpanID = "span-id-456"

		latencyMs := float64(input.Latency) / float64(time.Millisecond)

		expected := []zap.Field{
			zap.Int(StatusKey, input.Status),
			zap.String(MethodKey, input.Method),
			zap.String(PathKey, input.Path),
			zap.String(URIKey, input.URI),
			zap.Float64(LatencyKey, latencyMs),
			zap.String(RequestIDKey, input.RequestID),

			zap.String(TraceIDKey, input.TraceID),
			zap.String(SpanIDKey, input.SpanID),
		}

		actual := lf.BuildHTTPResponseFields(input)
		require.Equal(t, expected, actual)
	})
}

func TestLogFields_BuildSQLFields(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := NewLogFields(obsCfg)

	q := "SELECT 1\nFROM tbl"

	t.Run("引数/エラー/trace無しのケース", func(t *testing.T) {
		t.Parallel()

		s := SQLFieldsInput{
			Layer:    "layer",
			PkgName:  "pkg",
			FuncName: "fn",
			SpanName: "sn",
			Query:    q,
			Duration: 12 * time.Millisecond,
			Args:     nil,
			Err:      nil,
			TraceID:  "",
			SpanID:   "",
		}

		expected := []zap.Field{
			zap.String(LayerKey, s.Layer),
			zap.String(PackageKey, s.PkgName),
			zap.String(FunctionKey, s.FuncName),
			zap.String(SpanNameKey, s.SpanName),
			zap.String(RawQueryKey, q),
			zap.String(QueryCompactKey, "SELECT 1 FROM tbl"),
			zap.Float64(LatencyKey, float64(12)),
		}

		actual := lf.BuildSQLFields(s)
		require.Equal(t, expected, actual)
	})

	t.Run("引数/エラー/trace有りのケース", func(t *testing.T) {
		t.Parallel()

		err := fmt.Errorf("boom")
		args := []any{1, "a"}
		s := SQLFieldsInput{
			Layer:    "layer",
			PkgName:  "pkg",
			FuncName: "fn",
			SpanName: "sn",
			Query:    q,
			Duration: 12 * time.Millisecond,
			Args:     args,
			Err:      err,
			TraceID:  "tx",
			SpanID:   "sx",
		}

		expected := []zap.Field{
			zap.String(LayerKey, s.Layer),
			zap.String(PackageKey, s.PkgName),
			zap.String(FunctionKey, s.FuncName),
			zap.String(SpanNameKey, s.SpanName),
			zap.String(RawQueryKey, q),
			zap.String(QueryCompactKey, "SELECT 1 FROM tbl"),
			zap.Float64(LatencyKey, float64(12)),
			zap.Any(ArgsKey, args),
			zap.String(ArgsRawKey, fmt.Sprint(args)),
			zap.NamedError(InternalErrorKey, err),
			zap.String(TraceIDKey, "tx"),
			zap.String(SpanIDKey, "sx"),
		}

		actual := lf.BuildSQLFields(s)
		require.Equal(t, expected, actual)
	})
}

func TestLogFields_BuildObservabilityFields(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := NewLogFields(obsCfg)

	t.Run("基本項目とレイテンシのみ", func(t *testing.T) {
		t.Parallel()

		obs := ObservabilityFieldsInput{
			Layer:     "layer",
			PkgName:   "pkg",
			FuncName:  "fn",
			SpanEvent: "ev",
			SpanName:  "sn",
			Latency:   5 * time.Millisecond,
			TraceID:   "",
			SpanID:    "",
		}

		expected := []zap.Field{
			zap.String(SpanEventKey, obs.SpanEvent),
			zap.String(SpanNameKey, obs.SpanName),
			zap.String(LayerKey, obs.Layer),
			zap.String(PackageKey, obs.PkgName),
			zap.String(FunctionKey, obs.FuncName),
			zap.Float64(LatencyKey, float64(5)),
		}

		actual := lf.BuildObservabilityFields(obs)
		require.Equal(t, expected, actual)
	})

	t.Run("trace/span がある場合は追加される", func(t *testing.T) {
		t.Parallel()

		obs := ObservabilityFieldsInput{
			SpanEvent: "ev",
			SpanName:  "sn",
			Layer:     "layer",
			PkgName:   "pkg",
			FuncName:  "fn",
			Latency:   0,
			TraceID:   "tr",
			SpanID:    "sp",
		}

		expected := []zap.Field{
			zap.String(SpanEventKey, obs.SpanEvent),
			zap.String(SpanNameKey, obs.SpanName),
			zap.String(LayerKey, obs.Layer),
			zap.String(PackageKey, obs.PkgName),
			zap.String(FunctionKey, obs.FuncName),
			zap.String(TraceIDKey, obs.TraceID),
			zap.String(SpanIDKey, obs.SpanID),
		}

		actual := lf.BuildObservabilityFields(obs)
		require.Equal(t, expected, actual)
	})
}

func Test_appendTraceSpanFields(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	impl := NewLogFields(obsCfg).(*logFields)

	t.Run("trace/span が空なら追加されない", func(t *testing.T) {
		t.Parallel()
		base := []zap.Field{zap.String("a", "b")}
		got := impl.appendTraceSpanFields(base, "", "")
		require.Equal(t, base, got)
	})

	t.Run("trace/span があると追加される", func(t *testing.T) {
		t.Parallel()
		base := []zap.Field{zap.String("a", "b")}
		got := impl.appendTraceSpanFields(base, "t-1", "s-1")
		expected := []zap.Field{
			zap.String("a", "b"),
			zap.String(TraceIDKey, "t-1"),
			zap.String(SpanIDKey, "s-1"),
		}
		require.Equal(t, expected, got)
	})
}

func Test_latencyMs(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	impl := NewLogFields(obsCfg).(*logFields)

	t.Run("ミリ秒変換が正しい", func(t *testing.T) {
		t.Parallel()
		ms := impl.latencyMs(250 * time.Millisecond)
		require.InEpsilon(t, float64(250), ms, 0.01)
	})
}

func Test_buildCompactQuery(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	impl := NewLogFields(obsCfg).(*logFields)

	t.Run("改行/タブ/余白を詰める", func(t *testing.T) {
		t.Parallel()
		q := "SELECT  a,\n\t b  FROM\n table\tWHERE  x = 1"
		got := impl.buildCompactQuery(q)
		require.Equal(t, "SELECT a, b FROM table WHERE x = 1", got)
	})

	t.Run("空文字は空を返す", func(t *testing.T) {
		t.Parallel()
		require.Empty(t, impl.buildCompactQuery(""))
	})
}
