# redmetrics

English | [日本語](README.ja.md)

HTTP RED (Rate / Errors / Duration) metrics recorder and its Echo middleware.

## Role

Operational visibility into an HTTP service is built on the RED signals — request rate, error rate, and duration — and these must be captured uniformly for every request to be trustworthy. Isolating measurement as a middleware records one consistent set of metrics per request without each handler emitting its own, while keeping the label set deliberately low-cardinality so the resulting time series stay bounded and free of secrets.

## Notes

- `Middleware(rec Recorder)` returns the Echo middleware. It reads the final status inside the `c.Response().After` hook so that the status settled by the error handler / recovery is observed correctly, and it guards `Observe` with `sync.Once` so it is called exactly once per request (the `After` hook can fire multiple times for streaming / chunked responses).
- Known limitation: bodyless responses (e.g. `204 No Content`, `304 Not Modified`) never trigger a `Write`, so the `After` hook does not fire and the request is not measured. This is an accepted trade-off of observing the status after it is finalized.
- Ops endpoints (`/metrics`, etc.) are excluded from measurement via `ops.IsOpsPath`.
- Labels are limited to `method` / `route` / `status_code` / `status_class`; high-cardinality or sensitive values (raw path, query, user id, etc.) are never used. `route` is the Echo route pattern (e.g. `/users/:id`), falling back to `unknown` when unavailable; an unclassifiable status falls back to `unknown` as well.
- `Recorder` is the single-request recording interface. `PrometheusRecorder` implements it and also satisfies `prometheus.Collector`. `NewPrometheusRecorder()` has no side effects; `RegisterRecorder(reg, r)` registers it to a registry and ignores `AlreadyRegisteredError`.
- Metrics exposed: `http_server_requests_total` (counter) and `http_server_request_duration_seconds` (histogram, default buckets).
- `mock/` holds the mockgen-generated `Recorder` mock; do not edit it by hand.
