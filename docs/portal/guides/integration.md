## integration Directory

English | [日本語](README.ja.md)

This directory contains **integration tests**.

These tests start an Echo server and verify behavior through the **actual HTTP request path**, which cannot be fully covered by unit tests.

These tests are **not full system integration tests including DB or Infrastructure**, but rather **HTTP boundary tests**.

## Test Strategy

This project adopts the following test strategy.

```txt
Domain        → Unit Test
Usecase       → Unit Test
Controller    → Unit Test
Integration   → HTTP boundary test
```

Integration tests verify **the behavior of the entire HTTP path**.

Specifically, the following components are tested.

```txt
Router
↓
Middleware
↓
Handler
↓
Presenter / Response serialization
```

The following are **not covered by integration tests**:

- Domain logic
- Internal Usecase logic
- Repository implementations
- DB connections
- SQL execution

## Test Level Definition (Test Pyramid)

This project follows the **Test Pyramid** strategy.

```txt
        E2E (none)
      Integration
   Controller Unit
   Usecase Unit
     Domain Unit
```

Policy:

- Write most tests as **Domain / Usecase unit tests**
- Integration tests verify **only the HTTP boundary**
- **E2E tests are not handled in this repository**

This strategy provides the following benefits:

- Faster test execution
- Better test maintainability
- Easier root-cause analysis when tests fail

## Why Usecase Is Mocked

Integration tests **mock the Usecase layer**.

The reason is **to preserve layer boundaries**.

The purpose of integration tests is to verify the **HTTP layer**, limited to the following components.

```txt
Router
↓
Middleware
↓
Handler
↓
Response serialization
```

Usecase / Domain / Repository logic belongs to **unit tests**.

If the real Usecase implementation were used, the test scope would expand to include:

- DB
- Repository
- Domain

This would break the purpose of **HTTP boundary testing**.

## Integration Test Placement Policy

Integration tests are placed to verify **public API behavior**.

Example endpoints:

```txt
/health
/healthz
/ready
/version
/v1/...
```

In other words, integration tests target only:

```txt
Public HTTP APIs
```

Internal functions or detailed handler logic are verified by **Controller unit tests**.

## Why the integration Directory Exists

### Layer Separation

Unit tests target individual functions or handlers,  
while integration tests verify the **entire flow from router → middleware → handler → response serialization**.

Separating this directory clarifies **test purpose and granularity**.

```txt
internal/controller/handler/... → Handler Unit Test
internal/integration            → HTTP Integration Test
```

### Verification Close to Real Operation

Integration tests **send real HTTP requests**.

This allows verification of:

- Router binding
- Middleware application
- Request → DTO conversion
- Response serialization
- HTTP status codes

They can also be used for **CI/CD or smoke tests**.

## Scope of Integration Tests

Integration tests cover the following components.

```txt
Echo Router
↓
Middleware
↓
Handler
↓
Response
```

Usecase is **mocked**.

Reason:

The goal of integration tests is **HTTP boundary verification**,  
not application logic validation.

Example:

```txt
mock_user.NewMockUsecase
mock_healthcheck.NewMockUsecase
```

## Test Flow

Integration tests follow the steps below.

```txt
Echo.New()
↓
BindHandler
↓
StartServer
↓
HTTP Request
↓
Assert Response
```

Example:

```txt
echo.New()
↓
handler.BindHandler()
↓
StartServer()
↓
DoJSON()
↓
AssertJSONResponse()
```

## Functions Defined in integration_test.go

### `StartServer(t *testing.T, e *echo.Echo) *Server`

Starts Echo using `httptest.NewServer` and returns a test server.

Features:

- Server automatically stops via `t.Cleanup`
- Holds an internal HTTP client
- Provides helper functions `Do` / `DoJSON`

Example:

```go
e := echo.New()
handler.BindHandler(e)

srv := StartServer(t, e)
```

### `StopServer()`

Stops the server explicitly.

In most cases, the server stops automatically via `t.Cleanup` in `StartServer`.

### `Do(method, path string, reqBody io.Reader, contentType string, headers http.Header)`

Sends an HTTP request with arbitrary method, path, and body.

Features:

- Specify HTTP method
- Specify request body
- Specify headers
- Specify content-type

Internally uses `http.NewRequestWithContext`.

### `DoJSON(method, path string, reqBody any, headers http.Header)`

Shortcut function for JSON requests.

Features:

- Encodes `reqBody` as JSON
- Automatically sets `Content-Type: application/json`
- Internally calls `Do`

Example:

```go
actual := srv.DoJSON(http.MethodGet, "/health", nil, nil)
```

### `AssertJSONResponse[T any]`

Utility for verifying JSON responses.

Validation includes:

- HTTP Status Code = 200
- Content-Type = application/json
- JSON can be unmarshaled into type `T`

Example:

```go
AssertJSONResponse(t, gen.ResponseHealth{}, actual)
```

## Auth Test Helper

### `MakeAvailableUserID`

A helper that **simulates an authenticated user** in integration tests.

Internally it adds Echo middleware and sets authentication info using  
`ctxhelper.SetAuthnToEcho`.

Example:

```go
headers := MakeAvailableUserID(t, e, userID)
srv.DoJSON(http.MethodPost, "/v1/users", body, headers)
```

## Test Design Policy

Integration tests follow these principles.

### 1 Do Not Use Infrastructure

Integration tests must not use:

- DB
- SQL
- Repository

### 2 Mock the Usecase Layer

Integration tests **mock the Usecase layer**.

Reason:

```go
integration = HTTP boundary test
```

### 3 Use Real HTTP Requests

Handlers should not be called directly.

Instead, use:

```go
httptest.Server
```

### 4 Validate Using Response Types

Responses are verified using **OpenAPI generated types**.

```go
gen.ResponseV1Users
gen.ResponseHealth
gen.ResponseVersion
```
