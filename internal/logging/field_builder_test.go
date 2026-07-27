package logging

import (
	"testing"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogFields(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成したLogFieldBuilderが渡した設定を保持する", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			obsCfg := config.NewObservabilityConfig(cfg)
			osCfg := config.NewOperatingSystemConfig(cfg)

			builder := NewLogFields(obsCfg, osCfg)
			require.NotNil(t, builder)

			impl, ok := builder.(*logFieldBuilder)
			require.True(t, ok)
			assert.Same(t, obsCfg, impl.obsCfg)
			assert.Same(t, osCfg, impl.osCfg)
		})
	})
}

func Test_logFieldBuilder_BuildHTTPRequestFields(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	osCfg := config.NewOperatingSystemConfig(cfg)

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

		EventType: EventTypeStart,
		EventAt:   time.Now(),

		Host:          "example.com",
		Scheme:        "https",
		Proto:         "HTTP/1.1",
		UserAgent:     "Go-http-client/1.1",
		ContentType:   "application/json",
		ContentLength: 123,

		PathParams:  expectedPP,
		QueryParams: expectedQP,
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("trace_idとspan_idはbuilderでは付与されない（Loggerがctxから注入する）", func(t *testing.T) {
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
			assert.Equal(t, expected, actual)
		})

		t.Run("指定したイベント種別がevent_typeに反映される", func(t *testing.T) {
			t.Parallel()

			input := exampleInput
			input.EventType = EventTypePanic

			actual := lf.BuildHTTPRequestFields(input)
			assert.Contains(t, actual, String(EventTypeKey, EventTypePanic))
		})
	})
}

func Test_logFieldBuilder_BuildHTTPResponseFields(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	osCfg := config.NewOperatingSystemConfig(cfg)

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

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("trace_idとspan_idはbuilderでは付与されない（Loggerがctxから注入する）", func(t *testing.T) {
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
			assert.Equal(t, expected, actual)
		})
	})
}

func Test_logFieldBuilder_BuildSQLEndFields(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	osCfg := config.NewOperatingSystemConfig(cfg)

	lf := NewLogFields(obsCfg, osCfg)

	q := "SELECT 1\nFROM tbl"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("引数/エラー/trace無しの場合、基本項目のみを返す", func(t *testing.T) {
			t.Parallel()

			s := SQLFieldsEndInput{
				EventAt:  time.Now(),
				Layer:    "layer",
				PkgName:  "pkg",
				FuncName: "fn",
				SpanName: "sn",
				Query:    q,
				Latency:  12 * time.Millisecond,
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

			assert.Equal(t, expected, lf.BuildSQLEndFields(s))
		})

		t.Run("引数/エラー/parent_span_id有りの場合、内部エラーとparent_span_idが追加される", func(t *testing.T) {
			t.Parallel()

			err := xerrors.New("boom")
			args := []any{1, "a"}
			s := SQLFieldsEndInput{
				EventAt:      time.Now(),
				Layer:        "layer",
				PkgName:      "pkg",
				FuncName:     "fn",
				SpanName:     "sn",
				Query:        q,
				Latency:      12 * time.Millisecond,
				Args:         args,
				Err:          err,
				ParentSpanID: "px",
			}

			// trace_id / span_id は Logger が ctx から注入するため builder 出力には含まれない。
			// parent_span_id のみ builder が付与する（obs 有効時）。
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
				Int(QueryArgsCountKey, len(args)),
				Error(InternalErrorKey, err),
				String(ParentSpanIDKey, "px"),
			}

			assert.Equal(t, expected, lf.BuildSQLEndFields(s))
		})

		t.Run("引数のみ有りエラー無しの場合、引数件数のみ追加されエラーは追加されない", func(t *testing.T) {
			t.Parallel()

			s := SQLFieldsEndInput{
				EventAt:  time.Now(),
				Layer:    "layer",
				PkgName:  "pkg",
				FuncName: "fn",
				SpanName: "sn",
				Query:    q,
				Latency:  12 * time.Millisecond,
				Args:     []any{1, "a"},
			}

			keys := fieldKeys(lf.BuildSQLEndFields(s))
			assert.Contains(t, keys, QueryArgsCountKey)
			assert.NotContains(t, keys, InternalErrorKey)
		})

		t.Run("エラーのみ有り引数無しの場合、エラーのみ追加され引数件数は追加されない", func(t *testing.T) {
			t.Parallel()

			s := SQLFieldsEndInput{
				EventAt:  time.Now(),
				Layer:    "layer",
				PkgName:  "pkg",
				FuncName: "fn",
				SpanName: "sn",
				Query:    q,
				Latency:  12 * time.Millisecond,
				Err:      xerrors.New("boom"),
			}

			keys := fieldKeys(lf.BuildSQLEndFields(s))
			assert.Contains(t, keys, InternalErrorKey)
			assert.NotContains(t, keys, QueryArgsCountKey)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("obsが無効な場合、parent_span_idが有効でも追加されない", func(t *testing.T) {
			t.Parallel()

			disabledCfg := config.MockConfigForTest(t)
			disabledObsCfg := config.NewObservabilityConfig(disabledCfg)
			disabledObsCfg.SetObservabilityTracesExporter(t, "")
			disabledObsCfg.SetObservabilityMetricsExporter(t, "")
			disabledObsCfg.SetObservabilityLogsExporter(t, "")
			require.False(t, disabledObsCfg.Enabled())

			s := SQLFieldsEndInput{
				EventAt:      time.Now(),
				Layer:        "layer",
				PkgName:      "pkg",
				FuncName:     "fn",
				SpanName:     "sn",
				Query:        q,
				Latency:      12 * time.Millisecond,
				ParentSpanID: "p-1",
			}

			disabledLf := NewLogFields(disabledObsCfg, osCfg)
			keys := fieldKeys(disabledLf.BuildSQLEndFields(s))
			assert.NotContains(t, keys, ParentSpanIDKey)
		})
	})
}

// fieldKeys は、Field スライスからキー文字列の一覧を抽出するテストヘルパーです。
func fieldKeys(fs []*Field) []string {
	keys := make([]string, 0, len(fs))
	for _, f := range fs {
		keys = append(keys, f.key)
	}
	return keys
}

func Test_buildCompactQuery(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	osCfg := config.NewOperatingSystemConfig(cfg)
	impl, ok := NewLogFields(obsCfg, osCfg).(*logFieldBuilder)
	require.True(t, ok)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("改行/タブ/余白を詰める", func(t *testing.T) {
			t.Parallel()
			q := "SELECT  a,\n\t b  FROM\n table\tWHERE  x = 1"
			got := impl.buildCompactQuery(q)
			assert.Equal(t, "SELECT a, b FROM table WHERE x = 1", got)
		})

		t.Run("空文字は空を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, impl.buildCompactQuery(""))
		})
	})
}
