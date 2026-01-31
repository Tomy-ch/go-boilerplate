package logging

const (
	// HTTPレスポンス系ログのキー

	// EventTypeKey は、イベントタイプを表すログフィールドのキーです。
	EventTypeKey = "event_type"
	// EventTypeStart は、開始イベントを表すイベントタイプの値です。
	EventTypeStart = "start"
	// EventTypeEnd は、終了イベントを表すイベントタイプの値です。
	EventTypeEnd = "end"
	// EventAtKey は、イベント発生時刻を表すログフィールドのキーです。
	EventAtKey = "event_at"
	// EventTzKey は、イベント発生時刻のタイムゾーンを表すログフィールドのキーです。
	EventTzKey = "event_tz"

	// StatusKey は、HTTPステータスコードを表すログフィールドのキーです。
	StatusKey = "status"
	// MethodKey は、HTTPメソッドを表すログフィールドのキーです。
	MethodKey = "method"

	// URIKey は、HTTPリクエストURIを表すログフィールドのキーです。
	URIKey = "uri"
	// PathKey は、HTTPパスを表すログフィールドのキーです。
	PathKey = "path"
	// QueryParamsKey は、クエリパラメータを表すログフィールドのキーです。
	QueryParamsKey = "query_params"
	// PathParamsKey は、パスパラメータを表すログフィールドのキーです。
	PathParamsKey = "path_params"

	// UserAgentKey は、ユーザーエージェントを表すログフィールドのキーです。
	UserAgentKey = "user_agent"
	// HostKey は、ホスト名を表すログフィールドのキーです。
	HostKey = "host"
	// SchemeKey は、スキームを表すログフィールドのキーです。
	SchemeKey = "scheme"
	// ProtoKey は、プロトコルを表すログフィールドのキーです。
	ProtoKey = "proto"
	// RemoteIPKey は、リモートIPアドレスを表すログフィールドのキーです。
	RemoteIPKey = "remote_ip"
	// ContentTypeKey は、コンテンツタイプを表すログフィールドのキーです。
	ContentTypeKey = "content_type"
	// ContentLengthKey は、コンテンツ長を表すログフィールドのキーです。
	ContentLengthKey = "content_length"

	// LatencyKey は、HTTPリクエストのレイテンシを表すログフィールドのキーです。
	LatencyKey = "latency_ms"
	// RequestIDKey は、リクエストIDを表すログフィールドのキーです。
	RequestIDKey = "request_id"

	// エラー系ログのキー

	// ErrorCodeKey は、エラーコードを表すログフィールドのキーです。
	ErrorCodeKey = "error_code"
	// ErrorMessageKey は、エラーメッセージを表すログフィールドのキーです。
	ErrorMessageKey = "error_message"
	// ErrorDetails は、エラー詳細を表すログフィールドのキーです。
	ErrorDetails = "error_details"
	// InternalErrorKey は、内部エラーを表すログフィールドのキーです。
	InternalErrorKey = "internal_error"
	// InternalStackTraceKey は、内部エラーのスタックトレースを表すログフィールドのキーです。
	InternalStackTraceKey = "internal_stacktrace"

	// クエリ系ログのキー

	// RawQueryKey は、生のSQLクエリを表すログフィールドのキーです。
	RawQueryKey = "raw_query"
	// QueryCompactKey は、コンパクト化されたSQLクエリを表すログフィールドのキーです。
	QueryCompactKey = "query_compact"
	// QueryArgsKey は、SQLクエリの引数を表すログフィールドのキーです。
	QueryArgsKey = "args"
	// QueryArgsRawKey は、生のSQLクエリ引数を表すログフィールドのキーです。
	QueryArgsRawKey = "args_raw"

	// Job系ログのキー

	// JobNameKey は、ジョブ名を表すログフィールドのキーです。
	JobNameKey = "job_name"
	// JobArgsKey は、ジョブ引数を表すログフィールドのキーです。
	JobArgsKey = "job_args"
	// JobErrorKey は、ジョブエラーを表すログフィールドのキーです。
	JobErrorKey = "job_error"
	// JobResultKey は、ジョブ結果を表すログフィールドのキーです。
	JobResultKey = "job_result"

	// 可観測系ログのキー

	// TraceIDKey は、トレースIDを表すログフィールドのキーです。
	TraceIDKey = "trace_id"
	// SpanIDKey は、スパンIDを表すログフィールドのキーです。
	SpanIDKey = "span_id"
	// ParentSpanIDKey は、親スパンIDを表すログフィールドのキーです。
	ParentSpanIDKey = "parent_span_id"
	// SpanNameKey は、スパン名を表すログフィールドのキーです。
	SpanNameKey = "span_name"
	// LayerKey は、DDDのレイヤーを表すログフィールドのキーです。
	LayerKey = "layer"
	// PackageKey は、パッケージ名を表すログフィールドのキーです。
	PackageKey = "package"
	// FunctionKey は、関数名を表すログフィールドのキーです。
	FunctionKey = "function"
)
