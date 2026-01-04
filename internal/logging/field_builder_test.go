package logging

import (
	"fmt"
	"testing"
	"time"

	"boilerplate-go/internal/config"

	"github.com/stretchr/testify/require"
)

func TestNewLogFields(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)

	lf := NewLogFields(obsCfg, osCfg)
	require.NotNil(t, lf)
}

func TestLogFields_BuildHTTPRequestFields(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)

	lf := NewLogFields(obsCfg, osCfg)

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

		EventAt: time.Now(),

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

		expected := []*Field{
			String(EventTypeKey, EventTypeStart),
			Time(EventAtKey, exampleInput.EventAt),
			String(EventTzKey, osCfg.TimeZone()),

			String(MethodKey, exampleInput.Method),
			String(PathKey, exampleInput.Path),
			String(URIKey, exampleInput.URI),
			String(RemoteIPKey, exampleInput.RemoteIP),

			String(HostKey, exampleInput.Host),
			String(SchemeKey, exampleInput.Scheme),
			String(ProtoKey, exampleInput.Proto),
			String(UserAgentKey, exampleInput.UserAgent),
			String(ContentTypeKey, exampleInput.ContentType),
			Int64(ContentLengthKey, exampleInput.ContentLength),

			Any(QueryParamsKey, exampleInput.QueryParams),
			Any(PathParamsKey, exampleInput.PathParams),
		}

		actual := lf.BuildHTTPRequestFields(exampleInput)
		require.Equal(t, expected, actual)
	})

	t.Run("trace_idとspan_idが存在する場合、trace_idとspan_idのフィールドが追加される", func(t *testing.T) {
		t.Parallel()

		input := exampleInput
		input.TraceID = "trace-id-123"
		input.SpanID = "span-id-456"

		expected := []*Field{
			String(EventTypeKey, EventTypeStart),
			Time(EventAtKey, input.EventAt),
			String(EventTzKey, osCfg.TimeZone()),

			String(MethodKey, input.Method),
			String(PathKey, input.Path),
			String(URIKey, input.URI),
			String(RemoteIPKey, input.RemoteIP),

			String(HostKey, input.Host),
			String(SchemeKey, input.Scheme),
			String(ProtoKey, input.Proto),
			String(UserAgentKey, input.UserAgent),
			String(ContentTypeKey, input.ContentType),
			Int64(ContentLengthKey, input.ContentLength),

			Any(QueryParamsKey, input.QueryParams),
			Any(PathParamsKey, input.PathParams),

			String(TraceIDKey, input.TraceID),
			String(SpanIDKey, input.SpanID),
		}

		actual := lf.BuildHTTPRequestFields(input)
		require.Equal(t, expected, actual)
	})
}

func TestLogFields_BuildResponseFields(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)

	lf := NewLogFields(obsCfg, osCfg)

	exampleInput := HTTPResponseLogInput{
		EventAt:   time.Now(),
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

		expected := []*Field{
			String(EventTypeKey, EventTypeEnd),
			Time(EventAtKey, exampleInput.EventAt),
			String(EventTzKey, osCfg.TimeZone()),
			Float64(LatencyKey, latencyMs),

			Int(StatusKey, exampleInput.Status),
			String(MethodKey, exampleInput.Method),
			String(PathKey, exampleInput.Path),
			String(URIKey, exampleInput.URI),

			String(RequestIDKey, exampleInput.RequestID),
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

		expected := []*Field{
			String(EventTypeKey, EventTypeEnd),
			Time(EventAtKey, input.EventAt),
			String(EventTzKey, osCfg.TimeZone()),
			Float64(LatencyKey, latencyMs),

			Int(StatusKey, input.Status),
			String(MethodKey, input.Method),
			String(PathKey, input.Path),
			String(URIKey, input.URI),

			String(RequestIDKey, input.RequestID),

			String(TraceIDKey, input.TraceID),
			String(SpanIDKey, input.SpanID),
		}

		actual := lf.BuildHTTPResponseFields(input)
		require.Equal(t, expected, actual)
	})
}

func TestLogFields_BuildSQLStartFields(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)

	lf := NewLogFields(obsCfg, osCfg)

	t.Run("基本項目とレイテンシのみ", func(t *testing.T) {
		t.Parallel()

		s := SQLFieldsStartInput{
			EventAt:      time.Now(),
			Layer:        "layer",
			PkgName:      "pkg",
			FuncName:     "fn",
			SpanName:     "sn",
			TraceID:      "tx",
			SpanID:       "sx",
			ParentSpanID: "px",
		}

		expected := []*Field{
			String(EventTypeKey, EventTypeStart),
			Time(EventAtKey, s.EventAt),
			String(EventTzKey, osCfg.TimeZone()),
			String(LayerKey, s.Layer),
			String(PackageKey, s.PkgName),
			String(FunctionKey, s.FuncName),
			String(SpanNameKey, s.SpanName),
			String(TraceIDKey, s.TraceID),
			String(SpanIDKey, s.SpanID),
			String(ParentSpanIDKey, s.ParentSpanID),
		}

		actual := lf.BuildSQLStartFields(s)
		require.Equal(t, expected, actual)
	})

	t.Run("trace/spanが無い場合", func(t *testing.T) {
		t.Parallel()

		s := SQLFieldsStartInput{
			EventAt:  time.Now(),
			Layer:    "layer",
			PkgName:  "pkg",
			FuncName: "fn",
			SpanName: "sn",
		}

		expected := []*Field{
			String(EventTypeKey, EventTypeStart),
			Time(EventAtKey, s.EventAt),
			String(EventTzKey, osCfg.TimeZone()),
			String(LayerKey, s.Layer),
			String(PackageKey, s.PkgName),
			String(FunctionKey, s.FuncName),
			String(SpanNameKey, s.SpanName),
		}

		actual := lf.BuildSQLStartFields(s)
		require.Equal(t, expected, actual)
	})
}

func TestLogFields_BuildSQLEndFields(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)

	lf := NewLogFields(obsCfg, osCfg)

	q := "SELECT 1\nFROM tbl"

	t.Run("引数/エラー/trace無しのケース", func(t *testing.T) {
		t.Parallel()

		s := SQLFieldsEndInput{
			EventAt:  time.Now(),
			Layer:    "layer",
			PkgName:  "pkg",
			FuncName: "fn",
			SpanName: "sn",
			Query:    q,
			Latency:  12 * time.Millisecond,
			Args:     nil,
			Err:      nil,
			TraceID:  "",
			SpanID:   "",
		}

		expected := []*Field{
			String(EventTypeKey, EventTypeEnd),
			Time(EventAtKey, s.EventAt),
			String(EventTzKey, osCfg.TimeZone()),
			String(LayerKey, s.Layer),
			String(PackageKey, s.PkgName),
			String(FunctionKey, s.FuncName),
			String(SpanNameKey, s.SpanName),
			String(RawQueryKey, q),
			String(QueryCompactKey, "SELECT 1 FROM tbl"),
			Float64(LatencyKey, float64(12)),
		}

		actual := lf.BuildSQLEndFields(s)
		require.Equal(t, expected, actual)
	})

	t.Run("引数/エラー/trace有りのケース", func(t *testing.T) {
		t.Parallel()

		err := fmt.Errorf("boom")
		args := []any{1, "a"}
		s := SQLFieldsEndInput{
			EventAt:  time.Now(),
			Layer:    "layer",
			PkgName:  "pkg",
			FuncName: "fn",
			SpanName: "sn",
			Query:    q,
			Latency:  12 * time.Millisecond,
			Args:     args,
			Err:      err,
			TraceID:  "tx",
			SpanID:   "sx",
		}

		expected := []*Field{
			String(EventTypeKey, EventTypeEnd),
			Time(EventAtKey, s.EventAt),
			String(EventTzKey, osCfg.TimeZone()),
			String(LayerKey, s.Layer),
			String(PackageKey, s.PkgName),
			String(FunctionKey, s.FuncName),
			String(SpanNameKey, s.SpanName),
			String(RawQueryKey, q),
			String(QueryCompactKey, "SELECT 1 FROM tbl"),
			Float64(LatencyKey, float64(12)),
			Any(QueryArgsKey, args),
			String(QueryArgsRawKey, fmt.Sprint(args)),
			Error(InternalErrorKey, err),
			String(TraceIDKey, "tx"),
			String(SpanIDKey, "sx"),
		}

		actual := lf.BuildSQLEndFields(s)
		require.Equal(t, expected, actual)
	})
}

func TestLogFields_BuildObservabilityFields(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)

	lf := NewLogFields(obsCfg, osCfg)

	t.Run("基本項目とレイテンシのみ", func(t *testing.T) {
		t.Parallel()

		obs := ObservabilityFieldsInput{
			EventAt:   time.Now(),
			Layer:     "layer",
			PkgName:   "pkg",
			FuncName:  "fn",
			EventType: "ev",
			SpanName:  "sn",
			Latency:   5 * time.Millisecond,
			TraceID:   "",
			SpanID:    "",
		}

		expected := []*Field{
			String(EventTypeKey, obs.EventType),
			Time(EventAtKey, obs.EventAt),
			String(EventTzKey, osCfg.TimeZone()),
			String(SpanNameKey, obs.SpanName),
			String(LayerKey, obs.Layer),
			String(PackageKey, obs.PkgName),
			String(FunctionKey, obs.FuncName),
			Float64(LatencyKey, float64(5)),
		}

		actual := lf.BuildObservabilityFields(obs)
		require.Equal(t, expected, actual)
	})

	t.Run("trace/span がある場合は追加される", func(t *testing.T) {
		t.Parallel()

		obs := ObservabilityFieldsInput{
			EventAt:   time.Now(),
			EventType: "ev",
			SpanName:  "sn",
			Layer:     "layer",
			PkgName:   "pkg",
			FuncName:  "fn",
			Latency:   0,
			TraceID:   "tr",
			SpanID:    "sp",
		}

		expected := []*Field{
			String(EventTypeKey, obs.EventType),
			Time(EventAtKey, obs.EventAt),
			String(EventTzKey, osCfg.TimeZone()),
			String(SpanNameKey, obs.SpanName),
			String(LayerKey, obs.Layer),
			String(PackageKey, obs.PkgName),
			String(FunctionKey, obs.FuncName),
			String(TraceIDKey, obs.TraceID),
			String(SpanIDKey, obs.SpanID),
		}

		actual := lf.BuildObservabilityFields(obs)
		require.Equal(t, expected, actual)
	})
}

func Test_appendTraceSpanFields(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)
	impl := NewLogFields(obsCfg, osCfg).(*logFieldBuilder)

	t.Run("trace/spanが無い場合", func(t *testing.T) {
		t.Parallel()
		t.Run("parentSpanIDも無い場合", func(t *testing.T) {
			t.Parallel()
			base := []*Field{String("a", "b")}
			got := impl.appendTraceSpanFields(base, "", "", "")
			require.Equal(t, base, got)
		})

		t.Run("parentSpanIDのみ有る場合", func(t *testing.T) {
			t.Parallel()
			base := []*Field{String("a", "b")}
			got := impl.appendTraceSpanFields(base, "", "", "c")
			require.Equal(t, base, got)
		})
	})

	t.Run("trace/spanがある場合", func(t *testing.T) {
		t.Parallel()

		t.Run("parentSpanIDは無い場合", func(t *testing.T) {
			base := []*Field{String("a", "b")}
			got := impl.appendTraceSpanFields(base, "t-1", "s-1", "")
			expected := []*Field{
				String("a", "b"),
				String(TraceIDKey, "t-1"),
				String(SpanIDKey, "s-1"),
			}
			require.Equal(t, expected, got)
		})
		t.Run("parentSpanIDもある場合", func(t *testing.T) {
			base := []*Field{String("a", "b")}
			got := impl.appendTraceSpanFields(base, "t-1", "s-1", "p-1")
			expected := []*Field{
				String("a", "b"),
				String(TraceIDKey, "t-1"),
				String(SpanIDKey, "s-1"),
				String(ParentSpanIDKey, "p-1"),
			}
			require.Equal(t, expected, got)
		})
	})
}

func Test_buildCompactQuery(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)
	impl := NewLogFields(obsCfg, osCfg).(*logFieldBuilder)

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
