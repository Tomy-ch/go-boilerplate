//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE
package logging

import (
	"strings"
	"time"

	"go-boilerplate/internal/config"
)

type LogFieldBuilder interface {
	BuildHTTPRequestFields(req HTTPRequestLogInput) []*Field
	BuildHTTPResponseFields(resp HTTPResponseLogInput) []*Field
	BuildSQLStartFields(sql SQLFieldsStartInput) []*Field
	BuildSQLEndFields(sql SQLFieldsEndInput) []*Field
	BuildObservabilityFields(obs ObservabilityFieldsInput) []*Field
}

type logFieldBuilder struct {
	obsCfg *config.ObservabilityConfig
	osCfg  *config.OperatingSystemConfig
}

// HTTPRequestLogInput は、HTTPリクエストのログ出力用の入力情報をまとめた構造体です。
type HTTPRequestLogInput struct {
	// EventType は、イベント種別（start/error/panic）を表す。
	// リクエスト開始は start、エラー応答は error、パニック復旧は panic を指定する。
	EventType string
	EventAt   time.Time

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
	EventAt time.Time

	Method string
	Path   string
	URI    string
	Status int

	Latency time.Duration

	RequestID string
	TraceID   string
	SpanID    string
}

// SQLFieldsStartInput は、SQL ログ出力用の入力情報をまとめた構造体です。
type SQLFieldsStartInput struct {
	Layer    string
	PkgName  string
	FuncName string
	SpanName string

	EventAt time.Time

	TraceID      string
	SpanID       string
	ParentSpanID string
}

// SQLFieldsEndInput は、SQL ログ出力用の入力情報をまとめた構造体です。
type SQLFieldsEndInput struct {
	Layer    string
	PkgName  string
	FuncName string
	SpanName string

	EventAt time.Time

	Latency time.Duration

	Query string
	Args  []any
	Err   error

	TraceID      string
	SpanID       string
	ParentSpanID string
}

// ObservabilityFieldsInput は、オブザーバビリティ用のログフィールド生成の入力情報をまとめた構造体です。
type ObservabilityFieldsInput struct {
	Layer    string
	PkgName  string
	FuncName string
	SpanName string

	EventType string
	EventAt   time.Time

	Latency time.Duration

	TraceID      string
	SpanID       string
	ParentSpanID string
}

// NewLogFields は、ログフィールド生成用のLogFieldsインスタンスを生成します。
func NewLogFields(
	obsCfg *config.ObservabilityConfig,
	osCfg *config.OperatingSystemConfig,
) LogFieldBuilder {
	return &logFieldBuilder{
		obsCfg: obsCfg,
		osCfg:  osCfg,
	}
}

// BuildHTTPRequestFields は、HTTPリクエストの情報を含むFieldのスライスを生成します。
func (l *logFieldBuilder) BuildHTTPRequestFields(req HTTPRequestLogInput) []*Field {
	fields := append(l.buildEventHeader(req.EventType, req.EventAt),
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
	)

	return l.appendTraceSpanFields(fields, req.TraceID, req.SpanID, "")
}

// BuildHTTPResponseFields は、HTTPレスポンスの情報を含むFieldのスライスを生成します。
func (l *logFieldBuilder) BuildHTTPResponseFields(resp HTTPResponseLogInput) []*Field {
	fields := append(l.buildEventHeader(EventTypeEnd, resp.EventAt),
		DurationMs(LatencyKey, resp.Latency),

		Int(StatusKey, resp.Status),
		String(MethodKey, resp.Method),
		String(PathKey, resp.Path),
		String(URIKey, resp.URI),

		String(RequestIDKey, resp.RequestID),
	)

	return l.appendTraceSpanFields(fields, resp.TraceID, resp.SpanID, "")
}

// BuildSQLStartFields は、SQLの開始時点のログ出力用の Field スライスを構築します。
func (l *logFieldBuilder) BuildSQLStartFields(sql SQLFieldsStartInput) []*Field {
	fields := append(l.buildEventHeader(EventTypeStart, sql.EventAt),
		String(LayerKey, sql.Layer),
		String(PackageKey, sql.PkgName),
		String(FunctionKey, sql.FuncName),
		String(SpanNameKey, sql.SpanName),
	)

	return l.appendTraceSpanFields(fields, sql.TraceID, sql.SpanID, sql.ParentSpanID)
}

// BuildSQLEndFields は、SQLの終了時点のログ出力用の Field スライスを構築します。
func (l *logFieldBuilder) BuildSQLEndFields(sql SQLFieldsEndInput) []*Field {
	compact := l.buildCompactQuery(sql.Query)

	fields := append(l.buildEventHeader(EventTypeEnd, sql.EventAt),
		String(LayerKey, sql.Layer),
		String(PackageKey, sql.PkgName),
		String(FunctionKey, sql.FuncName),
		String(SpanNameKey, sql.SpanName),
		String(RawQueryKey, sql.Query),
		String(QueryCompactKey, compact),
		DurationMs(LatencyKey, sql.Latency),
	)

	if len(sql.Args) > 0 {
		fields = append(fields,
			Int(QueryArgsCountKey, len(sql.Args)),
		)
	}

	if sql.Err != nil {
		fields = append(fields, Error(InternalErrorKey, sql.Err))
	}

	return l.appendTraceSpanFields(fields, sql.TraceID, sql.SpanID, sql.ParentSpanID)
}

// BuildObservabilityFields は、オブザーバビリティ用のログフィールドを生成します。
func (l *logFieldBuilder) BuildObservabilityFields(obs ObservabilityFieldsInput) []*Field {
	fields := append(l.buildEventHeader(obs.EventType, obs.EventAt),
		String(SpanNameKey, obs.SpanName),
		String(LayerKey, obs.Layer),
		String(PackageKey, obs.PkgName),
		String(FunctionKey, obs.FuncName),
	)

	if obs.Latency > 0 {
		fields = append(fields,
			DurationMs(LatencyKey, obs.Latency),
		)
	}

	return l.appendTraceSpanFields(fields, obs.TraceID, obs.SpanID, obs.ParentSpanID)
}

// buildEventHeader は、全イベントログ共通の先頭フィールド
// （イベント種別・発生時刻・タイムゾーン）を構築する。
func (l *logFieldBuilder) buildEventHeader(eventType string, at time.Time) []*Field {
	return []*Field{
		String(EventTypeKey, eventType),
		Time(EventAtKey, at),
		String(EventTzKey, l.osCfg.TimeZone()),
	}
}

// appendTraceSpanFields は、obs が有効なとき trace/span を fields に追加する。
func (l *logFieldBuilder) appendTraceSpanFields(
	fields []*Field,
	traceID string,
	spanID string,
	parentSpanID string,
) []*Field {
	if !l.obsCfg.Enabled() || traceID == "" || spanID == "" {
		return fields
	}

	fields = append(fields,
		String(TraceIDKey, traceID),
		String(SpanIDKey, spanID),
	)

	if parentSpanID == "" {
		return fields
	}

	return append(fields,
		String(ParentSpanIDKey, parentSpanID),
	)
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
