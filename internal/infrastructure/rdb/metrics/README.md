# metrics

English | [日本語](README.ja.md)

`internal/infrastructure/rdb/metrics` is a package that **exposes pgxpool (PostgreSQL connection pool) statistics as Prometheus metrics**.

## Public API

|Function / Type|Description|
|---|---|
|`PoolStatsCollector`|Connection pool stats collector implementing `prometheus.Collector`|
|`New(db DatabaseDriver)`|Create a `PoolStatsCollector`|
|`RegisterPoolStatsCollector(c)`|Register collector with Prometheus registry (ignores duplicate registration)|

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

## Notes

- Obtains `pgxpool.Stat` from `DatabaseDriver.Stats()` and converts to metrics
- Safely skips duplicate registration by ignoring `prometheus.AlreadyRegisteredError`
