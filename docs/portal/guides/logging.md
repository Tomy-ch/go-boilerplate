# logging

English | [日本語](README.ja.md)

`internal/logging` provides a **structured logging foundation** used across the entire application.

This package is built on top of `zap`, but introduces an abstraction layer so that application code can **use logging without directly depending on zap**.

The primary goals are:

- Standardizing log formats
- Safely generating log fields
- Integrating with observability (trace/span)
- Improving testability
- Providing a framework-independent logging API

## Package Structure

```txt
internal/logging
├── logger.go
├── logger_core.go
├── field.go
├── field_builder.go
├── const.go
├── test_kit.go
└── mock/
```

The role of each file is as follows.

|File|Role|
|---|---|
|`logger.go`|Logger interface used by application code|
|`logger_core.go`|Implementation using `zap.Logger`|
|`field.go`|Type definition for log fields|
|`field_builder.go`|Generators for HTTP / SQL / Observability log fields|
|`const.go`|Log key definitions|
|`test_kit.go`|Logger / FieldBuilder for testing|

## Logger Interface

Application code only uses the **Logger interface**.

```go
type Logger interface {
    Debug(msg string, fields ...*Field)
    Info(msg string, fields ...*Field)
    Warn(msg string, fields ...*Field)
    Error(msg string, fields ...*Field)

    Named(name string) Logger
    CallerSkip(skip int) Logger
}
```

This design provides the following benefits:

- Encapsulates zap dependency internally
- Allows mocking in tests
- Enables logging implementation changes without affecting the application layer

## Logger Creation

The Logger is created depending on the application runtime mode.

```go
logger, err := logging.New(appCfg)
```

Internally, different logger configurations are used.

|Mode|Logger|
|---|---|
|production|JSON logger|
|development|console logger|

### Production Logger

```txt
Encoding: JSON
Level: Info
Stacktrace: Error and above
```

### Development Logger

```txt
Encoding: Console
Level: Debug
Stacktrace: Warn and above
```

## Field

Log fields are created using the `Field` type.

```go
logger.Info(
    "user created",
    logging.String("user_id", "123"),
    logging.Int("age", 20),
)
```

Supported field types:

|Function|Type|
|---|---|
|String|string|
|Strings|[]string|
|Int|int|
|Int64|int64|
|Float64|float64|
|Bool|bool|
|Error|error|
|Any|any|

The goals of this design:

- Prevent direct usage of `zap.Field`
- Ensure safe field generation
- Provide a unified logging API

## LogFieldBuilder

A component responsible for generating log fields for HTTP / SQL / Observability events.

```go
type LogFieldBuilder interface {
    BuildHTTPRequestFields(...)
    BuildHTTPResponseFields(...)
    BuildSQLStartFields(...)
    BuildSQLEndFields(...)
    BuildObservabilityFields(...)
}
```

Typical usage includes:

- HTTP access logging
- SQL logging
- trace/span logging

These fields enable **automatic structured logging generation**.

## HTTP Logging

HTTP request / response logs output the following fields.

Example (Request):

```txt
event_type=start
method=GET
path=/v1/users
remote_ip=...
trace_id=...
span_id=...
```

Example (Response):

```txt
event_type=end
status=200
latency_ms=12
trace_id=...
span_id=...
```

## SQL Logging

SQL logs are emitted as two events: **start** and **end**.

### SQL Start

```txt
event_type=start
layer=repository
span_name=FindUser
```

### SQL End

```txt
event_type=end
latency_ms=4
query=SELECT ...
args=[...]
```

Two types of query representations are logged:

```txt
raw_query
query_compact
```

## Observability Logging

Observability logs include trace and span information.

```txt
trace_id
span_id
parent_span_id
layer
package
function
```

If observability is disabled, these fields are not emitted.

## Test Kit

For testing, use `NewTestLogger`.

```go
logger := logging.NewTestLogger(t)
```

Features:

- Uses `zaptest.NewLogger`
- Outputs logs to `testing.T`
- No side effects

Test instance for `LogFieldBuilder`:

```go
logging.NewTestLogFieldBuilder(t)
```

## Design Policy

The logging package is designed based on the following policies.

### 1 Do Not Use zap Directly

Application code must not depend on:

```txt
zap.Logger
zap.Field
```

### 2 Wrap Fields

Log fields are generated using the `Field` type.

Reasons:

- Fix the field generation API
- Hide zap dependencies

### 3 Integrate Observability

Trace and span information are integrated within the logging layer.

```txt
trace_id
span_id
parent_span_id
```

### 4 Testability

Since Logger is an interface:

```txt
mockgen
```

can be used to generate mocks.

## Security Considerations

The following information **must not be logged**:

- passwords
- authentication tokens
- personal information

If such data must appear in logs, **masking should be applied**.
