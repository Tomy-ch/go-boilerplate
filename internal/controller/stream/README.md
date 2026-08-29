# stream

The generic SSE endpoint `GET /v1/streams/{destination}` — the whole Realtime Delivery transport:
the handler that decides **before the response is committed** whether a connection may start, and the
connection registry that runs everything after commit (admission, replay, catch-up, heartbeat,
control events, drain).

Design reference: `docs/design/realtime-delivery.md` §2.3 (connection lifecycle) and §3.1 (package
placement); ADR-0074 (query-ticket-stream-authentication).

## What happens on connect

| Order | Step | Where | Refusal |
| --- | --- | --- | --- |
| 1 | The query `ticket` is verified against the path `destination` and its bindings are placed in the `StreamGrant` context slot | `auth/` — the `StreamTicket` security scheme, run by the OpenAPI validator before any parameter validation | 401 (`UNAUTHORIZED`) — unknown, expired, revoked and wrong-destination tickets are indistinguishable |
| 2 | `after` / `Last-Event-ID` are checked against the spec pattern | OpenAPI validator | 400 (`BAD_REQUEST`) |
| 3 | The cursor is resolved: `Last-Event-ID` → `after` → the ticket's initial cursor | `cursor.go` | 400 (`INVALID_STREAM_CURSOR`) for a negative or out-of-range value |
| 4 | The cursor is checked against the replay floor | `usecase/realtime.CursorValidator` | 410 (`STREAM_CURSOR_EXPIRED`) when replay can no longer start there; 503 when the EventLog cannot be read |
| 5 | The verified `StreamRequest` is handed to `Streamer`, which indexes the connection and then verifies the ticket **once more** | `registry.go` | 503 (`SERVICE_UNAVAILABLE`) + `Retry-After` when the instance is at its connection cap, has no initial-replay slot within the bounded wait, or is draining |

Step 5's second verification exists because steps 1 and 4 are separated by external I/O. A revocation
that lands in that gap finds nothing to close — the connection is not in the registry's index yet —
and `AccessRevoker` invalidates tickets before it notifies, so re-checking after indexing settles it
either way: verified after the invalidation, the connection is refused; verified before it, the
connection was indexed before it too, and the notification finds it. Without this, a subject whose
access was withdrawn keeps receiving events until the connection lifetime expires, which
[ADR-0074](../../../docs/adr/0074-query-ticket-stream-authentication.md) names as a reason for
rejecting an alternative design.

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
Realtime DI module (`internal/di/module/realtime.go`), never in `di/module/controller.go`. That module is
not in the serve graph yet: it joins once a feature adapter needs it, which is #1416 and #1417.

## Runtime: what one connection does after commit

`Registry` is the instance's index of live connections. One value serves four callers, because all
four read the same index and splitting them would mean keeping separate locks in step: the `Streamer`
the handler calls, the `Waker` and `Revoker` the consumer engine (`controller/realtime`) hands
notifications to, and the `hook.Drainer` the serve lifecycle runs before HTTP shutdown. The index is
kept twice — by stream, because a wakeup names a stream, and by subject, because a revocation names a
subject. Neither is a feature word: the mechanism never learns which subject is a user and which is an
operator.

Each committed connection runs two goroutines that share nothing but channels:

| Goroutine | Owns | Does |
| --- | --- | --- |
| **fetcher** | the position read so far | initial replay, then re-reads on a wakeup or the jittered 30 s catch-up, taking a slot from the shared replay semaphore each time |
| **pump** (the handler's own goroutine) | the wire | writes events, control events and the 15 s heartbeat, resetting the write deadline before each write |

Three consequences worth knowing before changing any of it:

- **The initial-replay slot is taken before the response is committed**, and the fetcher keeps holding
  it through the first read. Waiting after commit would produce a connection that opened and then sat
  silent, which reads to a client as a hang rather than as backpressure.
- **A full send buffer closes the connection instead of dropping an event.** The buffer is one replay
  page (`ucrealtime.ReplayPageLimit`), and the events are still in the EventLog, so a reconnect replays
  from the client's own `Last-Event-ID`. Dropping instead would break the contiguous-prefix invariant
  the whole ordering chain rests on.
- **A read failure does not close the connection.** When the EventLog is unreachable the fetcher logs
  and waits for the next catch-up (design §2.6). Closing every connection on a dependency blip turns
  the recovery into a reconnect storm.

The `pump` takes a queued control event ahead of any buffered event, so a `STOP` never waits behind a
full buffer. Control events carry no SSE `id`, which is what keeps `Last-Event-ID` a pure function of
the business event stream.

### Stop-time budget

`Drain` refuses new connections, sends `RECONNECT` / `SERVER_DRAINING` to every open one, closes them,
and waits — for the shorter of the stop context and a fixed **10 seconds**. The work it waits on is one
control frame plus a close per connection, each already bounded by the 10 s write deadline, so the
budget is generous for what it covers. It is fixed rather than configurable because the rest of
`SHUTDOWN_TIMEOUT` belongs to the steps behind it: the consumer stop, the instance queue and
subscription teardown, and the HTTP shutdown. A drain that spent the whole budget would leave the queue
and lease behind for the orphan-cleanup job, which is the outcome the stop order exists to avoid
(design §2.5). This settles the open question handed over from #1414.

### Reason codes this package sends

`SERVER_DRAINING` (`RECONNECT`), `TEMPORARILY_OVERLOADED` (`RETRY_LATER`, best effort as the buffer
overflows), `AUTH_REFRESH_REQUIRED` (`REAUTHENTICATE`, at the 1 h lifetime), `AUTHORIZATION_REVOKED`
(`STOP`), and `CURSOR_TOO_OLD` (`RESYNC`, when a re-read comes back discontinuous).

`STREAM_RECOVERY_FAILED` is declared in the contract but **reserved** — this package never sends it. A
running connection survives an unreachable EventLog rather than failing, and a stream blocked by a dead
head row is visible to the relay, not to a connection.

## What is still elsewhere

- **Typed config.** `MaxConnections` and `ReplayConcurrency` are `Settings` fields whose zero values
  fall back to defaults, the same shape as `realtime.Settings` on the consumer engine. Exposing them as
  environment variables, and syncing `env/**`, is #1417.
- **Metrics.** Every close records a reason (`close_reason`) in a structured log, and the branches are
  separated so a counter can be attached to each. Registering the `realtime_*` metrics, the label rules
  and their architecture test is #1417, which also owns the span links — neither the delivery envelope
  nor the wakeup notification carries an origin trace today.
- **`OBS_TARGET_STATUS_CODES`.** 410 is present in `env/.env`, `.env.ci`, `.env.dast`, `.env.dev` and
  `.env.stg`, and absent only from `.env.prd`, so a `STREAM_CURSOR_EXPIRED` refusal is logged everywhere
  except production. Whether production should join them is an env-policy decision (`env/README.md`).

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
- The slot released when `commit` fails is a guard, not a path: `echo.Response.Flush` panics on a
  writer that cannot flush and swallows every other failure, so through Echo `commit` never returns
  an error and a real `ResponseWriter` always flushes. The behaviour is pinned one level down, on
  `sseWriter.commit` against a bare non-flushable writer, rather than by reaching for it through
  `Stream`.
- `sse_test.go`, `registry_test.go` and `connection_test.go` cover the post-commit half. Two rules
  shape them. **No test waits on real time**: the heartbeat, the lifetime and the catch-up all go
  through `clock.Sleeper`, and the tests supply one that blocks until the test names which tick to
  release, so a scheduling decision is asserted rather than timed. **Anything that flushes or sets a
  write deadline runs over `httptest.Server`**: `httptest.ResponseRecorder` implements neither, so a
  recorder-based test of the writer would assert against a writer the production path never uses.
- `internal/integration` drives the real validator → scheme → handler → registry chain over HTTP:
  401 / 400 / 410 / 200 for the pre-commit refusals, and for the connection itself the capacity cap,
  initial-replay saturation, resume by both `after` and `Last-Event-ID`, delivery by catch-up with no
  wakeup, revocation, and drain. `sse_client_test.go` is the Go reference client the design asks for —
  test-only, never a shipped SDK — and it is what pins the client contract: control events leave
  `Last-Event-ID` untouched, and `STOP` / `REAUTHENTICATE` / `RESYNC` make the client close before the
  server's EOF.
- Acceptance criterion 9 is split deliberately. That a saturated connection **closes** is pinned in
  `connection_test.go`, where the buffer can be filled deterministically; that **no event is lost** is
  pinned in `internal/integration`, by reconnecting and receiving what the first connection never read.
  Filling a real socket buffer over the loopback interface would make the first half a timing test.
