# buildinfo

English | [日本語](README.ja.md)

`internal/observability/metrics/buildinfo` is a package that **exposes the application's build / version / runtime information as a Prometheus info gauge (`app_build_info`)**.

It complements the `/version` HTTP endpoint by making the running version observable from the metrics backend (Prometheus / Grafana), which is useful for deployment confirmation, incident investigation, and rollback decisions.

## Public API

|Function / Type|Description|
|---|---|
|`Collector`|Build info collector implementing `prometheus.Collector`|
|`NewCollector(appCfg, bi)`|Resolve and normalize all label values once at wiring time and create a `Collector`|
|`Register(c)`|Register the collector with the Prometheus default registry (ignores duplicate registration)|

## Metric

|Item|Value|
|---|---|
|Name|`app_build_info`|
|Type|Gauge|
|Value|always `1` (the meaning is in the labels)|

### Labels

|Label|Example|Source|
|---|---|---|
|`service`|`go-boilerplate`|`config.ApplicationConfig.Name()`|
|`environment`|`production`|`config.ApplicationConfig.Env()`|
|`version`|`v1.5.0`|`system.BuildInfo.Version()`|
|`revision`|`abcdef1`|`system.BuildInfo.Revision()`|
|`build_date`|`2026-06-28T17:00:00Z`|`system.BuildInfo.BuildDate()`|
|`go_version`|`go1.24.4`|`runtime.Version()`|

## Design

- **Same source of truth as `/version`**: build values come from `system.BuildInfo`, so `/version` and `app_build_info` never disagree.
- **Resolved at wiring time**: build / version / runtime values do not change after startup, so all label values are resolved and normalized **once** in `NewCollector` and held. `Collect` only emits the held values via `MustNewConstMetric` (no per-scrape computation, unlike the mutable `pgxpool` pool stats collector).
- **Empty values become `unknown`**: empty label values are normalized to `unknown` at wiring time.

## Notes

- The following labels are intentionally **excluded** to avoid high cardinality and leaking secret / environment information: `hostname`, `pod_name`, `container_id`, `instance_id`, `git_branch`, `build_url`, `image_digest`, `full_image`, `token`, `registry`.
- This package does not call `os.Getenv` directly; the environment value comes from the immutable `config.ApplicationConfig`.
- Build info is never carried into the domain / usecase layers.
- Safely skips duplicate registration by ignoring `prometheus.AlreadyRegisteredError`.
