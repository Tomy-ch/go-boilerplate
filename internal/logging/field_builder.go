//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE
package logging

import (
	"fmt"
	"strings"
	"time"

	"boilerplate-go/internal/config"
)

type LogFieldBuilder interface {
	BuildHTTPRequestFields(req HTTPRequestLogInput) []*Field
	BuildHTTPResponseFields(resp HTTPResponseLogInput) []*Field
	BuildSQLFields(sql SQLFieldsInput) []*Field
	BuildObservabilityFields(obs ObservabilityFieldsInput) []*Field
}

type logFieldBuilder struct {
	obsCfg *config.ObservabilityConfig
}

// HTTPRequestLogInput は、HTTPリクエストのログ出力用の入力情報をまとめた構造体です。
type HTTPRequestLogInput struct {
	Method   string
	Path     string
	URI      string
	RemoteIP string

	Host          string
	Scheme        string
	Proto         string
	UserAgent     string
	ContentType   string
	ContentLength int64

	PathParams  map[string]string
	QueryParams map[string][]string

	TraceID string
	SpanID  string
}

// HTTPResponseLogInput は、HTTPレスポンスのログ出力用の入力情報をまとめた構造体です。
type HTTPResponseLogInput struct {
	Method    string
	Path      string
	URI       string
	Status    int
	Latency   time.Duration
	RequestID string
	TraceID   string
	SpanID    string
}

// SQLFieldsInput は、SQL ログ出力用の入力情報をまとめた構造体です。
type SQLFieldsInput struct {
	Layer    string
	PkgName  string
	FuncName string
	SpanName string

	Query    string
	Duration time.Duration
	Args     []any
	Err      error
	SpanID   string
	TraceID  string
}

// ObservabilityFieldsInput は、オブザーバビリティ用のログフィールド生成の入力情報をまとめた構造体です。
type ObservabilityFieldsInput struct {
	Layer     string
	PkgName   string
	FuncName  string
	SpanEvent string
	SpanName  string
	Latency   time.Duration

	SpanID  string
	TraceID string
}

// NewLogFields は、ログフィールド生成用のLogFieldsインスタンスを生成します。
func NewLogFields(
	obsCfg *config.ObservabilityConfig,
) LogFieldBuilder {
	return &logFieldBuilder{
		obsCfg: obsCfg,
	}
}

// BuildHTTPRequestFields は、HTTPリクエストの情報を含むFieldのスライスを生成します。
func (l *logFieldBuilder) BuildHTTPRequestFields(req HTTPRequestLogInput) []*Field {
	fields := []*Field{
		String(MethodKey, req.Method),
		String(PathKey, req.Path),
		String(URIKey, req.URI),
		String(RemoteIPKey, req.RemoteIP),

		String(HostKey, req.Host),
		String(SchemeKey, req.Scheme),
		String(ProtoKey, req.Proto),
		String(UserAgentKey, req.UserAgent),
		String(ContentTypeKey, req.ContentType),
		Int64(ContentLengthKey, req.ContentLength),

		Any(QueryParamsKey, req.QueryParams),
		Any(PathParamsKey, req.PathParams),
	}

	return l.appendTraceSpanFields(fields, req.TraceID, req.SpanID)
}

// BuildHTTPResponseFields は、HTTPレスポンスの情報を含むFieldのスライスを生成します。
func (l *logFieldBuilder) BuildHTTPResponseFields(resp HTTPResponseLogInput) []*Field {
	fields := []*Field{
		Int(StatusKey, resp.Status),
		String(MethodKey, resp.Method),
		String(PathKey, resp.Path),
		String(URIKey, resp.URI),
		Float64(LatencyKey, l.latencyMs(resp.Latency)),
		String(RequestIDKey, resp.RequestID),
	}

	return l.appendTraceSpanFields(fields, resp.TraceID, resp.SpanID)
}

// BuildSQLFields は、SQL ログ出力用の Field スライスを構築します。
func (l *logFieldBuilder) BuildSQLFields(s SQLFieldsInput) []*Field {
	rawQuery := s.Query
	compact := l.buildCompactQuery(rawQuery)

	fields := []*Field{
		String(LayerKey, s.Layer),
		String(PackageKey, s.PkgName),
		String(FunctionKey, s.FuncName),
		String(SpanNameKey, s.SpanName),
		String(RawQueryKey, rawQuery),
		String(QueryCompactKey, compact),
		Float64(LatencyKey, l.latencyMs(s.Duration)),
	}

	if len(s.Args) > 0 {
		fields = append(fields,
			Any(ArgsKey, s.Args),
			String(ArgsRawKey, fmt.Sprint(s.Args)),
		)
	}

	if s.Err != nil {
		fields = append(fields, Error(InternalErrorKey, s.Err))
	}

	return l.appendTraceSpanFields(fields, s.TraceID, s.SpanID)
}

// BuildObservabilityFields は、オブザーバビリティ用のログフィールドを生成します。
func (l *logFieldBuilder) BuildObservabilityFields(obs ObservabilityFieldsInput) []*Field {
	fields := []*Field{
		String(SpanEventKey, obs.SpanEvent),
		String(SpanNameKey, obs.SpanName),
		String(LayerKey, obs.Layer),
		String(PackageKey, obs.PkgName),
		String(FunctionKey, obs.FuncName),
	}

	if obs.Latency > 0 {
		fields = append(fields,
			Float64(LatencyKey, l.latencyMs(obs.Latency)),
		)
	}

	return l.appendTraceSpanFields(fields, obs.TraceID, obs.SpanID)
}

// appendTraceSpanFields は、obs が有効なとき trace/span を fields に追加する。
func (l *logFieldBuilder) appendTraceSpanFields(
	fields []*Field,
	traceID string,
	spanID string,
) []*Field {
	if !l.obsCfg.Enabled() || traceID == "" || spanID == "" {
		return fields
	}

	return append(fields,
		String(TraceIDKey, traceID),
		String(SpanIDKey, spanID),
	)
}

// latencyMs は、latency をミリ秒単位の float64 に変換します。
func (*logFieldBuilder) latencyMs(latency time.Duration) float64 {
	return float64(latency) / float64(time.Millisecond)
}

// buildCompactQuery は、SQL クエリを1行の短縮表現に変換します。
func (*logFieldBuilder) buildCompactQuery(q string) string {
	if q == "" {
		return ""
	}

	s := strings.ReplaceAll(q, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")

	s = strings.Join(strings.Fields(s), " ")

	return s
}
