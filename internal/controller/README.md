# Controller Layer Guide (`internal/controller`)

English | [日本語](README.ja.md)

## Role in Onion Architecture

- Acts as the **boundary between the external world (HTTP/CLI) and the application**
- Responsible for **protocol adaptation (Adapter)**, converting inputs into **application vocabulary (DTO / Value Objects)** and invoking the **Usecase layer**
- Performs **output formatting (Presenter)** by converting Usecase results into **OpenAPI response types**
- Maps exceptions (`error`) into **HTTP status codes** and **error codes** (`apperror` → Status)

> Key point: **Controllers must not contain business logic**.
> Their responsibility is limited to interpreting and formatting HTTP / CLI interactions.

## Directory Structure

```text
internal/controller/
├── handler/        # HTTP handlers (server entry points)
├── job/            # Job controllers (CLI entry points)
├── worker/         # Worker engine (message-queue entry points)
├── outbox/         # Outbox relay engine (polls the outbox and publishes)
├── server/         # Echo instance creation and server startup
├── httpstack/      # HTTP middleware stack
├── error/response/ # Error response generation
├── conv/           # OpenAPI-generated type → domain type boundary helpers
└── ctxhelper/      # Echo context helpers
```

## Subdirectory Roles

|Directory|Description|Details|
|---|---|---|
|`handler/`|Handlers that receive HTTP requests and delegate to Usecase|[README](handler/README.md)|
|`job/`|Job controllers invoked from CLI|[README](job/README.md)|
|`worker/`|Worker engine consuming a pull-ack message queue and dispatching to Usecase|[README](worker/README.md)|
|`outbox/`|Relay engine that periodically polls the outbox and publishes pending messages|—|
|`server/`|Echo instance initialization and DI lifecycle integration|[README](server/README.md)|
|`httpstack/`|Middleware stack (CORS, security, logging, auth, etc.)|[README](httpstack/README.md)|
|`error/response/`|Unified HTTP error response generation and apperror mapping|[README](error/response/README.md)|
|`conv/`|Boundary helpers converting OpenAPI-generated types into domain types|[README](conv/README.md)|
|`ctxhelper/`|Helpers for setting/getting values in Echo context|[README](ctxhelper/README.md)|

## Dependency Rules

```mermaid
flowchart TB
    Controller --> Usecase
    Controller --> apperror
    Controller --> Presenter

    Controller -. forbidden .-> Domain
    Controller -. forbidden .-> Infrastructure
    Controller -. forbidden .-> Database
```

Controllers access lower layers **only through Usecase**.

### Doc comments stay in HTTP vocabulary

The forbidden edges above govern doc comments as well. A handler doc comment states the HTTP-level
contract — what the endpoint returns, which status codes the failures map to, whether authentication
is required — and must not name tables, columns, SQL fragments, or where the transaction boundary
sits; those belong to the layer that owns them (see
[`internal/usecase/README.md`](../usecase/README.md) § Doc comments: interface vs implementation and
[`internal/infrastructure/README.md`](../infrastructure/README.md) § Doc comments may name technical
detail). A handler method is a method on the unexported `server`, so `revive` does not require a doc
comment on it: when the generated `ServerInterface` already carries the OpenAPI summary and there is
no HTTP-level detail worth adding, omit it rather than restating.

## Test Strategy

Handler tests mock the usecase and drive the handler through Echo (`testkit/testecho` + `testkit/testassert`); business logic lives in the usecase and is not re-tested here. Each handler test verifies:

- HTTP I/O conversion — request binding (path / query / body) → usecase input, and usecase output → response DTO / status
- request validation paths (OpenAPI / bind failures → 400 etc.)
- `apperror` → HTTP status mapping (the usecase error surfaced as the right status / code)
- middleware-supplied context — values the handler reads from context (auth principal / request id / idempotency)

Boundary-level HTTP wiring (Router → Middleware → Handler → Presenter) is covered separately by the `internal/integration` HTTP-boundary tests.
