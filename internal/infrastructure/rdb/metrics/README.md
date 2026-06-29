# metrics

English | [日本語](README.ja.md)

`internal/infrastructure/rdb/metrics` is a package that **exposes pgxpool (PostgreSQL connection pool) statistics as Prometheus metrics**.

## Role

The connection pool is the scarcest shared resource in the Infrastructure layer: when it saturates, requests pile up waiting to acquire a connection and latency degrades before any single query fails. This package exists to encapsulate that pool-observability concern — it translates the connection pool's runtime statistics snapshot into standard metrics so saturation signals (acquire waits, connection exhaustion, connection churn) become visible to operators and alerting. Isolating this conversion keeps the pool library's vendor-specific statistics API out of the rest of the system and surfaces pool health as a generic metrics-backend signal, letting capacity and timeout problems be caught before they turn into outages.

## Metrics List

Namespace: `pgxpool`

### Gauge (Current Values)

|Metric Name|Description|
|---|---|
|`pgxpool_acquired_conns`|Number of currently acquired connections|
|`pgxpool_idle_conns`|Number of currently idle connections|
|`pgxpool_total_conns`|Total connections in the pool|
|`pgxpool_constructing_conns`|Number of connections being constructed|
|`pgxpool_max_conns`|Maximum allowed connections|

### Counter (Cumulative Values)

|Metric Name|Description|
|---|---|
|`pgxpool_acquire_count_total`|Total successful connection acquires|
|`pgxpool_acquire_duration_seconds_total`|Total duration of connection acquires|
|`pgxpool_canceled_acquire_count_total`|Total canceled connection acquires|
|`pgxpool_empty_acquire_count_total`|Total acquires that created new connections due to empty pool|
|`pgxpool_new_conns_count_total`|Total new connections created|
|`pgxpool_max_lifetime_destroy_count_total`|Total connections destroyed due to max lifetime|
|`pgxpool_max_idle_destroy_count_total`|Total connections destroyed due to max idle time|
|`pgxpool_empty_acquire_wait_time_seconds_total`|Total wait time for acquires on empty pool|

## Query Metrics

While pool stats answer "is the pool saturated?", query metrics answer "which DB operation is slow / failing?". `NewQueryRecorder` returns a `driver.QueryRecorder` that the pgx query tracer (`driver.NewQueryTracer`) calls on every `TraceQueryEnd`, so Repository / QueryService SQL paths are instrumented transparently.

Namespace: `rdb` / Subsystem: `query`

|Metric Name|Type|Labels|Description|
|---|---|---|---|
|`rdb_query_duration_seconds`|Histogram|`query_name`, `operation`, `status`|DB query duration in seconds|
|`rdb_query_errors_total`|Counter|`query_name`, `operation`, `error_class`|Total failed DB queries|

Label semantics (kept low-cardinality, never carrying secrets):

- `query_name`: a stable app-managed name set via `driver.WithQueryName(ctx, "user.find_by_id")`. Unset → `unknown`.
- `operation`: normalized from the SQL leading token only (`select` / `insert` / `update` / `delete` / `begin` / `commit` / `rollback` / `copy` / `other`).
- `status`: `success` / `error`.
- `error_class`: `not_found` / `constraint` / `connection` / `timeout` / `unknown`, derived from `pgerror` normalization.

`pgx.ErrNoRows` is treated as `status=success` and is NOT counted in `rdb_query_errors_total` (it is a normal "not found" outcome decided by the upper layers).

The raw SQL text, bind values, table / column / constraint names, and PII are intentionally never used as labels. Use the query log / OTel trace for that level of detail.

## Notes

- Obtains `pgxpool.Stat` from `DatabaseDriver.Stats()` and converts to metrics
- Safely skips duplicate registration by ignoring `prometheus.AlreadyRegisteredError` (query metrics reuse the already-registered collector on duplicate init)
