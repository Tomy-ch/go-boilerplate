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
- `Method` is a **closed type** (struct-backed, unexported field): only the defined constants (`MethodGet` … `MethodDelete`) are constructible, so an invalid method string is rejected at compile time rather than at runtime (L2). Build requests with `NewRequest(method, downstream, url, opts...)` — `method` / `downstream` / `url` are required by the signature, and optional fields use `WithHeader` / `WithBody` / `WithIdempotencyKey` / `WithRetry` (the last sets `AllowRetry` + `IdempotencyKey` together so they can't drift apart).
- `Request` is an **immutable value object** (L2): all fields are unexported, so it is constructible only through `NewRequest` + `With*` options and readable only through getters (`Downstream()` / `Method()` / `URL()` / `Header()` / `Body()` / `IdempotencyKey()` / `AllowRetry()`). This makes invalid states (e.g. missing `downstream`, or `AllowRetry` without `IdempotencyKey`) unrepresentable from outside the package; the runtime guard inside `Do` remains as in-package defense-in-depth.
- Per-`Downstream` resilience is resolved by `Registry`: each gateway contributes a `DownstreamProfile` to the `httpclient_profiles` fx group, and unregistered keys fall back to `DefaultProfile`.
- Retry safety is method-aware: idempotent methods (GET / PUT / DELETE) are always retry-safe; non-idempotent methods (POST / PATCH) are retried only when `AllowRetry` is set, which then requires `IdempotencyKey`.
- Retryable outcomes are 5xx / 429 / transport failures; 4xx / success / context cancellation are not retried. Backoff is exponential with full jitter, honoring a `Retry-After` header when present.
- A per-downstream retry budget (token bucket) caps retry amplification, and a per-downstream circuit breaker (closed / half-open / open) fails fast under sustained downstream faults.
- Two timeout layers are enforced: a per-attempt timeout and an overall timeout; a backoff wait that would overrun the overall deadline aborts the retry.
- Security defaults: redirects are not followed (`http.ErrUseLastResponse`, SSRF surface reduction), response bodies are read only up to `MaxResponseBytes`, error messages are redacted of query / userinfo / fragment, and trace propagation / private-network access are opt-out per downstream.
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
