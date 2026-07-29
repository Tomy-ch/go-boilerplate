# Direct Dependency Inventory

A **living inventory** of this project's direct third-party dependencies, grouped by the
single responsibility each fulfils. Unlike an ADR, this list is *expected to drift* with
`go.mod` — it is a reference, not an immutable record.

- The **policy** for adopting a dependency (one responsibility = one concern) is a decision:
  [`ADR-0068`](../adr/0068-library-selection-policy.md).
- **Bridge / instrumentation** libraries that straddle two upstreams are accepted as bounded
  exceptions to that policy: [`ADR-0069`](../adr/0069-bridge-instrumentation-exceptions.md).

> Keep this table in sync with `go.mod` (`require` block, non-indirect entries). Versions
> below are a snapshot and are not authoritative — `go.mod` is.

## Direct dependencies by responsibility

| Area | Library | Responsibility |
| --- | --- | --- |
| Web / API | `labstack/echo/v5` | HTTP web framework (see [ADR-0017](../adr/0017-echo-http-framework.md)) |
| Web / API | `oapi-codegen/echo-v5-middleware` | OpenAPI request-validation middleware for Echo |
| Web / API | `oapi-codegen/runtime` | Runtime support for oapi-codegen generated code |
| Web / API | `getkin/kin-openapi` | OpenAPI 3 document model / loader |
| Config | `caarlos0/env/v11` | Env var → struct decoding |
| Config | `joho/godotenv` | Loading `.env` files |
| Database | `jackc/pgx/v5` | PostgreSQL driver |
| Database | `golang-migrate/migrate/v4` | Schema migration runner |
| Errors / utils | `cockroachdb/errors` | Error wrapping with stack traces |
| Errors / utils | `google/uuid` | UUID generation (UUIDv7, see [ADR-0031](../adr/0031-uuidv7-identifiers.md)) |
| Errors / utils | `golang.org/x/crypto` | Cryptographic primitives |
| Errors / utils | `golang.org/x/sync` | Concurrency primitives (errgroup, etc.) |
| DI / logging / CLI | `go.uber.org/fx` | Dependency injection container (see [ADR-0032](../adr/0032-uber-fx-di.md)) |
| DI / logging / CLI | `go.uber.org/zap` | Structured logging |
| DI / logging / CLI | `spf13/cobra` | CLI command framework |
| Testing | `go.uber.org/mock` | Mock generation runtime |
| Testing | `stretchr/testify` | Assertions |
| Messaging / worker | `aws/aws-sdk-go-v2` | AWS API client core (worker adapter, opt-in — see [ADR-0044](../adr/0044-sqs-adapter-opt-in.md)) |
| Messaging / worker | `aws/aws-sdk-go-v2/service/sqs` | SQS client (pull-ack worker) |
| Metrics exposition | `prometheus/client_golang` | Prometheus-format metrics endpoint + custom collectors |
| Metrics exposition | `prometheus/client_model` | Prometheus metric data model (shared types) |
| Observability (otel core) | `go.opentelemetry.io/otel` (+ `trace` / `metric`) | OpenTelemetry API |
| Observability (otel core) | `go.opentelemetry.io/otel/sdk` | OpenTelemetry trace SDK |
| Observability (otel core) | `go.opentelemetry.io/otel/sdk/metric` | OpenTelemetry metric SDK |
| Observability (otel core) | `go.opentelemetry.io/otel/sdk/log` | OpenTelemetry log SDK |
| Observability (otel core) | `exporters/otlp/otlptrace/{otlptracehttp,otlptracegrpc}` | OTLP trace exporters (built from `OBS_*` config) |
| Observability (otel core) | `exporters/otlp/otlpmetric/{otlpmetrichttp,otlpmetricgrpc}` | OTLP metric exporters (built from `OBS_*` config) |
| Observability (otel core) | `exporters/otlp/otlplog/{otlploghttp,otlploggrpc}` | OTLP log exporters (built from `OBS_*` config) |
| Observability (otel core) | `contrib/instrumentation/runtime` | Go runtime metrics |

The otel core group includes pre-v1.0 (`v0.x`) modules (the OTLP log exporters and
`sdk/log`), but each couples to a **single** upstream (OpenTelemetry itself), not two, so
they are in-policy and not treated as exceptions. The OTLP exporters are constructed
explicitly from typed `OBS_*` config rather than via `contrib/exporters/autoexport`
(see [ADR-0062](../adr/0062-config-driven-observability-gating.md)).

## Bridge / instrumentation exceptions

These stand between **two independently-versioned upstreams** (a framework/library ×
OpenTelemetry), so they fall outside "one concern, one upstream" and are accepted as bounded
exceptions per [ADR-0069](../adr/0069-bridge-instrumentation-exceptions.md).

| Library | Coupling | Role |
| --- | --- | --- |
| `labstack/echo-opentelemetry` | Echo `MiddlewareFunc` × otel trace | Root server span per request (status / path normalization / W3C propagation) |
| `contrib/instrumentation/net/http/otelhttp` | `net/http` `RoundTripper`/`Handler` × otel trace | Outbound/inbound `net/http` instrumentation (client transport + handler spans) |
| `exaring/otelpgx` | pgx `QueryTracer` × otel trace | SQL query spans via the pgx tracer hook |
| `contrib/bridges/otelzap` | zap `zapcore.Core` × otel/log | Bridges zap records into OTel log records for OTLP export |

> Per-library version and prod-LOC investigations that inform the fork-cost bound live in the
> ADR's history if needed; they are point-in-time and deliberately not maintained here.

## Notes

- Previously the dependency table lived inline in `docs/decisions.md`; it drifted (the
  `net/http/otelhttp` instrumentation and `otel/sdk/log` SDK were missing). Splitting the
  *inventory* (this file) from the *policy* ([ADR-0068](../adr/0068-library-selection-policy.md))
  is why: the immutable decision no longer carries a list that must track `go.mod`.
