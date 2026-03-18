# app error

English | [日本語](README.ja.md)

The `apperror` package defines **protocol-agnostic error classifications shared across the application**.

This package can be referenced from **Domain / Usecase / Controller / Infrastructure layers**,  
and provides **base error categories for application errors** in a protocol-independent way.

HTTP status codes and API response formats are **not handled here**.  
They are **translated in the Controller layer**.

## Basic Policy

- Can be referenced from Domain / Usecase / Controller / Infrastructure
- Only define **application-wide base error categories**
- Must not contain HTTP status codes or response formats
- Designed to work with `xerrors.Is` / `xerrors.As`

Examples

- `ErrInvalidArgument`
- `ErrNotFound`
- `ErrConflict`

## Usage Rules

When returning errors, it is **strongly recommended to wrap them with an `apperror` base category**.

Reasons

- Enables error classification via `xerrors.Is`
- Allows the Controller layer to convert errors into HTTP statuses
- Preserves original errors for logging and tracing

## Recommended Error Wrapping Pattern

Errors should **always wrap a base error category**.

```go
// Wrap a domain error with an application error category
if err != nil {
    return xerrors.Wrap(apperror.ErrConflict, "failed to create user")
}
```

The Controller layer determines the category using `xerrors.Is`.

```go
// Map application error to HTTP status
if xerrors.Is(err, apperror.ErrNotFound) {
    return lookupErrorMetaByHTTPStatus(http.StatusNotFound)
}
```

## Translating Infrastructure Errors

In the Infrastructure layer, it is recommended to **translate external dependency errors into `apperror`**.

Reasons

- Converts DB / external API errors into application-level vocabulary
- Prevents upper layers from depending on DB-specific errors

Example

```go
// Translate database error into an application error
if xerrors.As(err, &sql.ErrNoRows) {
    switch pgErr.Code {
    case "23505": // unique constraint violation
        return xerrors.Wrap(apperror.ErrConflict, err.Error())
    default:
        return xerrors.Wrap(apperror.ErrInternal, err.Error())
    }
}
```

Typically this translation occurs in:

- Repository
- Infrastructure Adapter

## HTTP Error Conversion (Controller Layer)

The `apperror` package **does not know about HTTP**.

Conversion to HTTP status codes is the **responsibility of the Controller layer**.

In this boilerplate, the Controller's `errorhandler` middleware performs a **two-step conversion**.

```txt
apperror
   ↓
HTTP Status
   ↓
Error Meta (status / code / message)
```

Example

```go
case xerrors.Is(err, apperror.ErrNotFound):
    return lookupErrorMetaByHTTPStatus(http.StatusNotFound)
```

`lookupErrorMetaByHTTPStatus` returns **HTTP error metadata** containing:

- HTTP Status
- Error Code
- Message

This approach provides the following advantages:

- Domain / Usecase remain HTTP-independent
- Error messages are centrally managed in the Controller
- API specification changes do not require modifications to Domain logic

## Error Handling in Job / CLI

`apperror` can be used not only in HTTP Controllers but also in **Job / CLI Controllers**.

In job execution, the common flow is:

- Log the error
- The Runner decides the exit code

```txt
Usecase
    return apperror.ErrUnavailable

Job Controller
    log error

Job Runner
    exit code decision
```

## Adding New Error Categories

New error categories should **not be added casually**.

Evaluation criteria

```txt
OK

- Occurs across multiple use cases
- Represents a common concept across the application

NG

- Used only by a single use case
- Added solely due to HTTP status requirements
```

If a new category is added, document the following in the README:

- Background
- Usage scenarios
- Corresponding HTTP status

## Mapping Table

|app error definition|Meaning / Usage|HTTP Status|
|----------------------|----------------|-------------|
|`ErrInvalidArgument`|Invalid argument (syntactically valid but semantically invalid)|400 Bad Request|
|`ErrUnauthenticated`|Authentication failure (e.g., not logged in)|401 Unauthorized|
|`ErrPermissionDenied`|Insufficient permissions|403 Forbidden|
|`ErrNotFound`|Target resource does not exist|404 Not Found|
|`ErrConflict`|Conflict (unique constraint violation, concurrent update conflict, etc.)|409 Conflict|
|`ErrValidation`|Domain / usecase validation failure|422 Unprocessable Entity|
|`ErrTooManyRequests`|Too many requests|429 Too Many Requests|
|`ErrInternal`|Unexpected internal error|500 Internal Server Error|
|`ErrUnimplemented`|Not implemented / unsupported functionality|501 Not Implemented|
|`ErrUnavailable`|Temporarily unavailable (external dependency failure, etc.)|503 Service Unavailable|
