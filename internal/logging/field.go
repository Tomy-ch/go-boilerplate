//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE
package logging

import (
	"fmt"
	"strings"
	"time"

	"boilerplate-go/internal/config"

	"go.uber.org/zap"
)

type LogFields interface {
	BuildHTTPRequestFields(req HTTPRequestLogInput) []zap.Field
	BuildHTTPResponseFields(resp HTTPResponseLogInput) []zap.Field
	BuildSQLFields(sql SQLFieldsInput) []zap.Field
	BuildObservabilityFields(obs ObservabilityFieldsInput) []zap.Field
}

type logFields struct {
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
) LogFields {
	return &logFields{
		obsCfg: obsCfg,
	}
}

// BuildHTTPRequestFields は、HTTPリクエストの情報を含むzap.Fieldのスライスを生成します。
func (l *logFields) BuildHTTPRequestFields(req HTTPRequestLogInput) []zap.Field {
	fields := []zap.Field{
		zap.String(MethodKey, req.Method),
		zap.String(PathKey, req.Path),
		zap.String(URIKey, req.URI),
		zap.String(RemoteIPKey, req.RemoteIP),

		zap.String(HostKey, req.Host),
		zap.String(SchemeKey, req.Scheme),
		zap.String(ProtoKey, req.Proto),
		zap.String(UserAgentKey, req.UserAgent),
		zap.String(ContentTypeKey, req.ContentType),
		zap.Int64(ContentLengthKey, req.ContentLength),

		zap.Any(QueryParamsKey, req.QueryParams),
		zap.Any(PathParamsKey, req.PathParams),
	}

	return l.appendTraceSpanFields(fields, req.TraceID, req.SpanID)
}

// BuildHTTPResponseFields は、HTTPレスポンスの情報を含むzap.Fieldのスライスを生成します。
func (l *logFields) BuildHTTPResponseFields(resp HTTPResponseLogInput) []zap.Field {
	fields := []zap.Field{
		zap.Int(StatusKey, resp.Status),
		zap.String(MethodKey, resp.Method),
		zap.String(PathKey, resp.Path),
		zap.String(URIKey, resp.URI),
		zap.Float64(LatencyKey, l.latencyMs(resp.Latency)),
		zap.String(RequestIDKey, resp.RequestID),
	}

	return l.appendTraceSpanFields(fields, resp.TraceID, resp.SpanID)
}

// BuildSQLFields は、SQL ログ出力用の zap.Field スライスを構築します。
func (l *logFields) BuildSQLFields(s SQLFieldsInput) []zap.Field {
	rawQuery := s.Query
	compact := l.buildCompactQuery(rawQuery)

	fields := []zap.Field{
		zap.String(LayerKey, s.Layer),
		zap.String(PackageKey, s.PkgName),
		zap.String(FunctionKey, s.FuncName),
		zap.String(SpanNameKey, s.SpanName),
		zap.String(RawQueryKey, rawQuery),
		zap.String(QueryCompactKey, compact),
		zap.Float64(LatencyKey, l.latencyMs(s.Duration)),
	}

	if len(s.Args) > 0 {
		fields = append(fields,
			zap.Any(ArgsKey, s.Args),
			zap.String(ArgsRawKey, fmt.Sprint(s.Args)),
		)
	}

	if s.Err != nil {
		fields = append(fields, zap.NamedError(InternalErrorKey, s.Err))
	}

	return l.appendTraceSpanFields(fields, s.TraceID, s.SpanID)
}

// BuildObservabilityFields は、オブザーバビリティ用のログフィールドを生成します。
func (l *logFields) BuildObservabilityFields(obs ObservabilityFieldsInput) []zap.Field {
	fields := []zap.Field{
		zap.String(SpanEventKey, obs.SpanEvent),
		zap.String(SpanNameKey, obs.SpanName),
		zap.String(LayerKey, obs.Layer),
		zap.String(PackageKey, obs.PkgName),
		zap.String(FunctionKey, obs.FuncName),
	}

	if obs.Latency > 0 {
		fields = append(fields,
			zap.Float64(LatencyKey, l.latencyMs(obs.Latency)),
		)
	}

	return l.appendTraceSpanFields(fields, obs.TraceID, obs.SpanID)
}

// appendTraceSpanFields は、obs が有効なとき trace/span を fields に追加する。
func (l *logFields) appendTraceSpanFields(
	fields []zap.Field,
	traceID string,
	spanID string,
) []zap.Field {
	if !l.obsCfg.Enabled() || traceID == "" || spanID == "" {
		return fields
	}

	return append(fields,
		zap.String(TraceIDKey, traceID),
		zap.String(SpanIDKey, spanID),
	)
}

// latencyMs は、latency をミリ秒単位の float64 に変換します。
func (*logFields) latencyMs(latency time.Duration) float64 {
	return float64(latency) / float64(time.Millisecond)
}

// buildCompactQuery は、SQL クエリを1行の短縮表現に変換します。
func (*logFields) buildCompactQuery(q string) string {
	if q == "" {
		return ""
	}

	s := strings.ReplaceAll(q, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")

	s = strings.Join(strings.Fields(s), " ")

	return s
}
