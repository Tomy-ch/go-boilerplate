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
}
```

This design provides:

- Encapsulation of zap dependency
- Ability to replace with mocks in tests
- No impact on application layer even if logging implementation changes

## Logger Creation

Logger is created according to the application runtime mode.

```go
logger, err := logging.New(appCfg)
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
|Error|error|
|Any|any|

Purpose of this design

- Prevent direct usage of zap.Field
- Ensure safe field generation
- Unify API

## LogFieldBuilder

A component that consolidates log field generation for HTTP / SQL / Observability.

```go
type LogFieldBuilder interface {
    BuildHTTPRequestFields(...)
    BuildHTTPResponseFields(...)
    BuildSQLStartFields(...)
    BuildSQLEndFields(...)
    BuildObservabilityFields(...)
}
```

Use cases

- HTTP access logs
- SQL logs
- trace/span logs

Automatically generates **structured logs**.

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

## Security Considerations

Be careful not to output the following information in logs.

- passwords
- authentication tokens
- personal information

If necessary, apply **masking processing**.
