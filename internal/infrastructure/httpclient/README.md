# httpclient

English | [日本語](README.ja.md)

`internal/infrastructure/httpclient` provides a **resilient substrate for outbound HTTP** (retry / circuit breaker / budget / tracing), consumed by semantic interface implementations such as gateways and publishers.

## Architectural Position

```mermaid
flowchart TB
    subgraph "Usecase Layer"
        GW["exchangerate.Gateway interface"]
        PUB["publisher.Publisher interface"]
    end
    subgraph "Infrastructure Layer"
        GWImpl["webapi/exchangerate impl"]
        PUBImpl["publisher impl"]
        Sub["httpclient.Client substrate"]
    end

    GWImpl -. implements .-> GW
    PUBImpl -. implements .-> PUB
    GWImpl --> Sub
    PUBImpl --> Sub
```

This package does **not** implement a domain / usecase boundary interface. It is a driver-level substrate (the HTTP counterpart of `rdb/driver`), consumed by `webapi/` and `publisher/`. Callers express only intent (`Request`) and result (`Response`); status interpretation, `apperror` mapping, timeout / retry / budget / breaker / observability all stay inside the substrate.

## Design Policy

- net/http is never exposed: own types (`Method` / `Header` / `Request` / `Response` / `Downstream`) are public, and status interpretation + `apperror` mapping are closed inside the substrate (the HTTP version of `pgerror.NormalizeError`).
- Per-`Downstream` resilience is resolved by `Registry`: each gateway contributes a `DownstreamProfile` to the `httpclient_profiles` fx group, and unregistered keys fall back to `DefaultProfile`.
- Retry safety is method-aware: idempotent methods (GET / PUT / DELETE) are always retry-safe; non-idempotent methods (POST / PATCH) are retried only when `AllowRetry` is set, which then requires `IdempotencyKey`.
- Retryable outcomes are 5xx / 429 / transport failures; 4xx / success / context cancellation are not retried. Backoff is exponential with full jitter, honoring a `Retry-After` header when present.
- A per-downstream retry budget (token bucket) caps retry amplification, and a per-downstream circuit breaker (closed / half-open / open) fails fast under sustained downstream faults.
- Two timeout layers are enforced: a per-attempt timeout and an overall timeout; a backoff wait that would overrun the overall deadline aborts the retry.
- Security defaults: redirects are not followed (`http.ErrUseLastResponse`, SSRF surface reduction), response bodies are read only up to `MaxResponseBytes`, error messages are redacted of query / userinfo / fragment, and trace propagation / private-network access are opt-out per downstream. The post-DNS dial guard (in `internal/observability`) always blocks link-local (cloud metadata `169.254.169.254`) / unspecified / bogon-reserved ranges (TEST-NETs, Future Use, IETF Assignments, Benchmarking, IPv6 Documentation), and blocks loopback / private (RFC1918, ULA) / CGNAT (RFC 6598 `100.64.0.0/10`) unless `AllowPrivateNetwork` is set.
- Transport / status events are normalized into `apperror` sentinels (`ErrUnavailable` / `ErrCanceled` / `ErrInvalidArgument` etc.); callers branch on sentinels, never on raw status.

## DI Registration

Registered by the `httpclient` module in `internal/di/module/httpclient.go`. Each downstream contributes a `Profile` to the `httpclient_profiles` group, which the `Registry` aggregates.

```go
fx.Module("httpclient",
    fx.Provide(
        provideHTTPClientRegistry,
        httpclient.New,
    ),
)
```
