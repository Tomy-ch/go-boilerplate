# logging

English | [日本語](README.ja.md)

`internal/logging` provides a **structured logging foundation** used across the entire application.

This package is based on `zap`, while providing an abstraction layer that allows application code to handle logging **without directly depending on zap**.

The main purposes are as follows.

- Standardization of log formats
- Safe generation of log fields
- Integration with observability (trace/span)
- Ensuring testability
- Providing a framework-independent logging API

## Package Structure

```txt
internal/logging
├── logger.go
├── logger_core.go
├── stacktrace_core.go
├── field.go
├── field_builder.go
├── const.go
├── test_kit.go
└── mock/
```

The role of each file is as follows.

|File|Role|
|---|---|
|`logger.go`|Logger interface used by application|
|`logger_core.go`|Implementation of zap.Logger|
|`stacktrace_core.go`|zapcore.Core wrapper that converts the auto-attached `Entry.Stack` into a line array for JSON output|
|`field.go`|Type of log fields|
|`field_builder.go`|Generation of HTTP / SQL / Observability log fields|
|`const.go`|Log key definitions|
|`test_kit.go`|Logger / FieldBuilder for testing|

## Logger Interface

Application code uses only the **Logger interface**.

```go
type Logger interface {
    Debug(msg string, fields ...*Field)
    Info(msg string, fields ...*Field)
    Warn(msg string, fields ...*Field)
    Error(msg string, fields ...*Field)

    Named(name string) Logger
    CallerSkip(skip int) Logger
    ConvertFields(fields []*Field) []zap.Field
}
```

`ConvertFields` converts a `*Field` slice to a `zap.Field` slice. Primarily used for framework integration where `zap.Field` is required internally.

This design provides:

- Encapsulation of zap dependency
- Ability to replace with mocks in tests
- No impact on application layer even if logging implementation changes

## Logger Creation

Logger is created according to the application runtime mode.

```go
logger, err := logging.New(appCfg)
```

`New` selects the appropriate logger based on the mode of `config.ApplicationConfig`.

You can also create loggers individually:

```go
logger, err := logging.NewProductionLogger()
logger, err := logging.NewDevelopmentLogger()
```

Internally, the following loggers are used.

|Mode|Logger|
|---|---|
|production|JSON logger|
|development|console logger|

### Production Logger

- Encoding: JSON
- Level: Info
- Stacktrace: Error and above

### Development Logger

- Encoding: Console
- Level: Debug
- Stacktrace: Warn and above

## Field

Log fields are created using the `Field` type.

```go
logger.Info(
    "user created",
    logging.String("user_id", "123"),
    logging.Int("age", 20),
)
```

Supported types

|Function|Type|
|---|---|
|String|string|
|Strings|[]string|
|Int|int|
|Int64|int64|
|Float64|float64|
|Bool|bool|
|Time|time.Time (converted to RFC3339Nano string)|
|DurationMs|time.Duration (converted to float64 in milliseconds)|
|Error|error|
|Stacktrace|error (converted to stack trace lines as []string)|
|Any|any|

Purpose of this design

- Prevent direct usage of zap.Field
- Ensure safe field generation
- Unify API

## LogFieldBuilder

A component that consolidates log field generation for HTTP / SQL / Observability.

```go
type LogFieldBuilder interface {
    BuildHTTPRequestFields(req HTTPRequestLogInput) []*Field
    BuildHTTPResponseFields(resp HTTPResponseLogInput) []*Field
    BuildSQLStartFields(sql SQLFieldsStartInput) []*Field
    BuildSQLEndFields(sql SQLFieldsEndInput) []*Field
    BuildObservabilityFields(obs ObservabilityFieldsInput) []*Field
}
```

Creation

```go
lf := logging.NewLogFields(obsCfg, osCfg)
```

Accepts `config.ObservabilityConfig` and `config.OperatingSystemConfig` to control trace/span field attachment and timezone information.

Use cases

- HTTP access logs
- SQL logs
- trace/span logs

Automatically generates **structured logs**.

### Input Structs

Each Build method receives a dedicated input struct.

|Struct|Use|Key Fields|
|---|---|---|
|`HTTPRequestLogInput`|HTTP request log|Method, Path, URI, RemoteIP, Host, Scheme, Proto, UserAgent, ContentType, ContentLength, PathParams, QueryParams|
|`HTTPResponseLogInput`|HTTP response log|Method, Path, URI, Status, Latency, RequestID|
|`SQLFieldsStartInput`|SQL start log|Layer, PkgName, FuncName, SpanName|
|`SQLFieldsEndInput`|SQL end log|Layer, PkgName, FuncName, SpanName, Latency, Query, Args, Err|
|`ObservabilityFieldsInput`|Observability log|Layer, PkgName, FuncName, SpanName, EventType, Latency|

All input structs share `EventAt` (event timestamp) and `TraceID` / `SpanID` / `ParentSpanID` (trace information) as common fields.

## HTTP Logging

HTTP request / response logs output the following fields.

Example (Request)

- `event_type=start`
- `method=GET`
- `path=/v1/users`
- `remote_ip=...`
- `trace_id=...`
- `span_id=...`

Example (Response)

- `event_type=end`
- `status=200`
- `latency_ms=12`
- `trace_id=...`
- `span_id=...`

## SQL Logging

SQL logs are output as two events: **start / end**.

### SQL Start

- `event_type=start`
- `layer=repository`
- `span_name=FindUser`

### SQL End

- `event_type=end`
- `latency_ms=4`
- `query=SELECT ...`
- `args=[...]`

Queries are output in two formats.

- `raw_query`
- `query_compact`

## Observability Logging

Observability logs include trace/span information.

- `trace_id`
- `span_id`
- `parent_span_id`
- `layer`
- `package`
- `function`

If observability is disabled, these are not output.

## Test Kit

In tests, use `NewTestLogger`.

```go
logger := logging.NewTestLogger(t)
```

Features

- `zaptest.NewLogger`
- Outputs test logs to `testing.T`
- No side effects

Test instance for LogFieldBuilder

```go
logging.NewTestLogFieldBuilder(t)
```

## Design Policy

This logging package is designed based on the following policies.

### 1 Do not use zap directly

Application code does not depend on `zap.Logger`, `zap.Field`.

### 2 Wrap Field

Log fields use the `Field` type.

Reason

- Fix the field generation API
- Hide zap dependency

### 3 Integrate Observability

trace / span information is integrated at the logging layer.

- `trace_id`
- `span_id`
- `parent_span_id`

### 4 Testability

Since Logger is an interface, it can be mocked using `mockgen`.

## Log Key Constants

Log keys defined in `const.go`.

### HTTP

|Constant|Key|
|---|---|
|`EventTypeKey`|`event_type`|
|`EventTypeStart`|`start`|
|`EventTypeEnd`|`end`|
|`EventAtKey`|`event_at`|
|`EventTzKey`|`event_tz`|
|`StatusKey`|`status`|
|`MethodKey`|`method`|
|`URIKey`|`uri`|
|`PathKey`|`path`|
|`QueryParamsKey`|`query_params`|
|`PathParamsKey`|`path_params`|
|`UserAgentKey`|`user_agent`|
|`HostKey`|`host`|
|`SchemeKey`|`scheme`|
|`ProtoKey`|`proto`|
|`RemoteIPKey`|`remote_ip`|
|`ContentTypeKey`|`content_type`|
|`ContentLengthKey`|`content_length`|
|`LatencyKey`|`latency_ms`|
|`RequestIDKey`|`request_id`|

### Error

|Constant|Key|
|---|---|
|`ErrorCodeKey`|`error_code`|
|`ErrorMessageKey`|`error_message`|
|`ErrorDetails`|`error_details`|
|`InternalErrorKey`|`internal_error`|
|`InternalStackTraceKey`|`internal_stacktrace`|

### Query

|Constant|Key|
|---|---|
|`RawQueryKey`|`raw_query`|
|`QueryCompactKey`|`query_compact`|
|`QueryArgsCountKey`|`args_count`|

### Job

|Constant|Key|
|---|---|
|`JobNameKey`|`job_name`|
|`JobArgsKey`|`job_args`|
|`JobErrorKey`|`job_error`|
|`JobResultKey`|`job_result`|

### Observability

|Constant|Key|
|---|---|
|`TraceIDKey`|`trace_id`|
|`SpanIDKey`|`span_id`|
|`ParentSpanIDKey`|`parent_span_id`|
|`SpanNameKey`|`span_name`|
|`LayerKey`|`layer`|
|`PackageKey`|`package`|
|`FunctionKey`|`function`|

## Security Considerations

Be careful not to output the following information in logs.

- passwords
- authentication tokens
- personal information

If necessary, apply **masking processing**.
