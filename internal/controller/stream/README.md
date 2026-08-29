# stream

The handler for the generic SSE endpoint `GET /v1/streams/{destination}` — the part of the Realtime
Delivery transport that decides **before the response is committed** whether a connection may start.
The part that runs after commit (the connection registry, admission, replay, heartbeat, control
events) is the `Streamer` seam this package declares and Phase 6 implements.

Design reference: `docs/design/realtime-delivery.md` §2.3 (connection lifecycle) and §3.1 (package
placement); ADR-0074 (query-ticket-stream-authentication).

## What happens on connect

| Order | Step | Where | Refusal |
| --- | --- | --- | --- |
| 1 | The query `ticket` is verified against the path `destination` and its bindings are placed in the `StreamGrant` context slot | `auth/` — the `StreamTicket` security scheme, run by the OpenAPI validator before any parameter validation | 401 (`UNAUTHORIZED`) — unknown, expired, revoked and wrong-destination tickets are indistinguishable |
| 2 | `after` / `Last-Event-ID` are checked against the spec pattern | OpenAPI validator | 400 (`BAD_REQUEST`) |
| 3 | The cursor is resolved: `Last-Event-ID` → `after` → the ticket's initial cursor | `cursor.go` | 400 (`INVALID_STREAM_CURSOR`) for a negative or out-of-range value |
| 4 | The cursor is checked against the replay floor | `usecase/realtime.CursorValidator` | 410 (`STREAM_CURSOR_EXPIRED`) when replay can no longer start there; 503 when the EventLog cannot be read |
| 5 | The verified `StreamRequest` is handed to `Streamer` | Phase 6 | — |

Every refusal is an ordinary HTTP error answered by the shared error handler; nothing here writes a
response body. The ticket's raw value never appears in an error, a log field or a span attribute —
`httpstack/redaction` strips it from the request URI and query before they are logged.

## Why the handler is not strict-server

ADR-0014 (oapi-codegen-strict-server) generates every other handler with `strict-server`, whose
glue marshals one typed response object per call. An SSE response is a long-lived stream of events
written with explicit flushes, deadlines and an in-band close; it has no single return value, so the
`v1/streams` tag is generated with `echo5-server` only and the handler receives the `echo.Context`.
This is the one exception the design reference names.

## Placement

The package lives beside `handler/`, not under it: it is transport for a mechanism, not a feature
resource, and `internal/architest/realtime_isolation_test.go` forbids it from importing any
`internal/domain/<feature>` or `internal/usecase/<feature>`. `BindHandler` is registered by the
Realtime DI module (`internal/di/module/realtime.go`), never in `di/module/controller.go`; the module joins the
serve graph when the `Streamer` and a feature adapter exist (Phase 6).

## What Phase 6 must settle before the handler is wired

- **The Realtime DI module (`realtimeModule()`) registers `BindHandler`, but that module is not in the serve graph yet**, so on every environment the operation answers 401 (`ErrUnauthorizedSchemeUnsupported`) until Phase 6 provides the `Streamer` and a feature adapter binds the module. `TestBindHandlerDIParity` covers this package (declared ⇔ invoked in `realtime.go`), so a `BindHandler` left out of the module is red; the module itself being out of the app graph is the documented Phase 5 boundary.
- **The shared request timeout and write timeout cut a stream**: `timeout.Middleware` (Pre priority 2) applies `SERVER_REQUEST_TIMEOUT` (60s by default) to every request and `http.Server.WriteTimeout` (65s) closes the connection after that. Neither has a per-path exclusion today; the design's "stream path is excluded from the request timeout" has to be built before an SSE response can outlive a minute.
- **The fan-out hands its notifications to sinks this package's connection registry has to implement** (Phase 7, #1414): `controller/realtime.WakeupSink.Wake(streamID, upTo)` — wake the connections on that stream whose cursor is below `upTo` (idempotent; duplicates are normal, and coalescing across receive batches is the registry's) — and `controller/realtime.RevocationSink.Revoke(subject, destination)` — close that subject's connections with `STOP`. Both are called synchronously on the consumer loop, so they only mark or signal. The serve lifecycle stops in the order drain → consumer stop → inbox teardown → HTTP shutdown (`internal/di/server/hook`), and the registry supplies the drain as a `hook.Drainer` participant (refuse new connections, send `RECONNECT` / `SERVER_DRAINING`, wait for the connections to close) from the realtime DI module.
- **410 is not in `OBS_TARGET_STATUS_CODES`** in any environment, so a `STREAM_CURSOR_EXPIRED` refusal reaches the client but no log — adding it is an env-policy decision (`env/README.md`).

## Sub-packages

| Package | Role |
| --- | --- |
| `auth/` | `SchemeAuthenticator` for the `StreamTicket` security scheme: reads the parameter the scheme declares, calls `TicketVerifier.Verify`, writes `ctxhelper.SetStreamGrant` |
| `gen/` | oapi-codegen output for the `v1/streams` tag (types + non-strict echo server) |

## Test Strategy

- `stream_handler_test.go` pins the pre-commit contract with a mocked `CursorValidator` and a stub
  `Streamer`: cursor precedence, the error class and `code` of every refusal, and that `Streamer`
  is never reached after a refusal.
- `cursor_test.go` pins the decimal parsing at its edges (`0`, `MaxInt64`, overflow, sign, leading
  zeros) — the spec pattern admits 19-digit values, so overflow is reachable.
- `auth/stream_ticket_test.go` pins that the scheme name matches `openapi.yaml`, that the verifier
  receives the request context and the path destination, and that a rejected ticket leaves the slot
  empty.
- `internal/integration` drives the real validator → scheme → handler chain over HTTP for 401 / 400
  / 410 / 200.
