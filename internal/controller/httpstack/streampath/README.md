# streampath

`Is(path)` — whether a request is the SSE stream endpoint (`GET /v1/streams/{destination}`), by the
`/v1/streams/` prefix. It exists so that middleware written for request/response-shaped traffic can
step aside for a response that stays open for an hour.

## What consults it, and why

| Consumer | Without the exclusion |
| --- | --- |
| `timeout` | `SERVER_REQUEST_TIMEOUT` (60 s) cancels the request context, so every stream would be cut a minute in |
| `logging` | One access-log line is written when the response completes — an hour late, with a duration that measures the connection rather than a request |
| `redmetrics` | The RED histogram would take that same hour as request latency and skew every percentile the API is judged by |

The connection's own bounds replace them: a per-write deadline (10 s), the heartbeat, and the
maximum connection lifetime, all owned by `controller/stream`.

## What must not consult it

**`oapi/skipper`**, and OpenAPI validation generally. The stream endpoint's `ticket` is verified by
the `StreamTicket` security scheme, which the OpenAPI validator runs — skip validation on this path
and every connection is admitted unauthenticated. This is why the exclusion is a separate predicate
rather than another entry in `ops.IsOpsPath`: that one is consulted by `oapi/skipper` too, and ops
paths are exempt from validation precisely because they have no OpenAPI definition. The stream
endpoint does have one, and needs it enforced.
