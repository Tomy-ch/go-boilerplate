# streampath

Two predicates, so that middleware written for request/response-shaped traffic can step aside for a
response that stays open for an hour — without also stepping aside for the refusals that never
became one.

- `Is(path)` — the request is aimed at the SSE stream endpoint (`GET /v1/streams/{destination}`), by
  the `/v1/streams/` prefix. Answerable **before** the handler runs.
- `IsCommittedStream(header)` — the response actually committed as a stream, by its
  `text/event-stream` content type. Answerable only **after**.

## What consults which, and why

| Consumer | Predicate | Without the exclusion |
| --- | --- | --- |
| `timeout` | `Is` | `SERVER_REQUEST_TIMEOUT` (60 s) cancels the request context, so every stream would be cut a minute in. The budget is set before the handler runs, so this one has to be decided on the path |
| `logging` | `IsCommittedStream` | The response log line is written when the connection closes — an hour late, with a duration that measures the connection rather than a request. The *request* log line still goes out at connect time |
| `redmetrics` | `IsCommittedStream` | The RED histogram would take that same hour as request latency and skew every percentile the API is judged by |

The connection's own bounds replace the request budget: a per-write deadline (10 s), the heartbeat,
and the maximum connection lifetime, all owned by `controller/stream`. The store calls it makes are
bounded by `dynamodbclient.CallTimeout`, because the request budget no longer covers them.

**Why the observability pair use the response and not the path.** The stream endpoint refuses
connections with 401, 400, 410 and 503, and those are ordinary responses that finish in
milliseconds. Excluding them by path would delete exactly the signals worth watching — a ticket
brute-force reads as a burst of 401s, capacity exhaustion as a burst of 503s — and neither would
appear in a log or a metric. The reason for the exclusion ("a connection's length is not a request
latency") only applies once the connection exists.

## What must not consult it

**`oapi/skipper`**, and OpenAPI validation generally. Neither predicate belongs there. The stream endpoint's `ticket` is verified by
the `StreamTicket` security scheme, which the OpenAPI validator runs — skip validation on this path
and every connection is admitted unauthenticated. This is why the exclusion is a separate predicate
rather than another entry in `ops.IsOpsPath`: that one is consulted by `oapi/skipper` too, and ops
paths are exempt from validation precisely because they have no OpenAPI definition. The stream
endpoint does have one, and needs it enforced.
