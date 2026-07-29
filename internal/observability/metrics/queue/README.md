# queue

English | [日本語](README.ja.md)

`internal/observability/metrics/queue` is a package that **exposes worker broker-queue backlog
(queue depth / DLQ count) as Prometheus metrics**.

## Role

A worker can be perfectly healthy (processing, low error rate) while its queue still backs up —
producers outpace consumers, or messages pile up in the DLQ. The engine's processed / failed /
retry counters cannot show that; they describe throughput, not backlog. This package fills the gap
by scraping the optional `worker.QueueStatsProvider` capability (implemented by broker adapters
such as SQS) and publishing the current backlog as a gauge, so saturation is observable alongside
throughput.

The engine never depends on `QueueStatsProvider`; only this collector does. Depth is pulled at
scrape time through the capability, keeping broker-specific APIs out of the engine.

## Metrics List

Namespace: `worker`, Subsystem: `queue`

### Gauge (Current Values)

|Metric Name|Labels|Description|
|---|---|---|
|`worker_queue_depth`|`worker`, `adapter`, `queue`, `state`|Approximate backlog by state. `queue` is `source` / `dlq`; `state` is `visible` / `not_visible` / `delayed`.|

### Counter (Cumulative Values)

|Metric Name|Labels|Description|
|---|---|---|
|`worker_queue_stats_collection_failures_total`|`worker`, `adapter`, `queue`|Number of scrape-time collection failures. `queue` is `unknown` because a single `QueueStats` call does not distinguish source vs DLQ.|

## Labels

Allowed: `worker`, `adapter`, `queue`, `state`. Queue URL / ARN, message id, receipt handle and any
high-cardinality or secret-bearing value are intentionally **never** used as labels — worker name
and adapter kind are sufficient for aggregation and alerting.

## Notes

- Targets are collected via the `worker.queue_stats_targets` DI group; with no targets the
  collector emits nothing.
- SQS attribute values are **approximate** — read `worker_queue_depth` as a backlog trend, not an
  exact count.
- Scrape calls the broker API directly (no cache). A TTL cache may be added later to bound API
  rate / cost.
- Each scrape is bounded by a 5s timeout (`collectTimeout`) covering all targets; without it an
  unresponsive broker would let scrape goroutines accumulate and exhaust resources. Targets are
  collected **concurrently** under that shared deadline, so one slow provider cannot consume the
  budget and drag healthy queues into false collection failures.
- Safely skips duplicate registration by ignoring `prometheus.AlreadyRegisteredError`.
