# webapi

English | [日本語](README.ja.md)

`internal/infrastructure/webapi` is the **parent subsystem for external Web API gateways** — each leaf implements a usecase `boundary.Gateway` interface on top of the `httpclient` resilient substrate.

## Architectural Position

```mermaid
flowchart TB
    subgraph "Usecase Layer (boundary)"
        IF["&lt;service&gt;.Gateway interface"]
    end
    subgraph "Infrastructure Layer"
        Impl["webapi/&lt;service&gt; gateway"]
        Sub["httpclient.Client substrate"]
    end

    Impl -. implements .-> IF
    Impl --> Sub
```

Each leaf under `webapi/` implements a semantic gateway interface defined in `internal/usecase/boundary/<service>` and delegates transport to the `httpclient` substrate. Usecase / Domain depend only on the boundary, never on HTTP details. Leaf implementations do not carry their own README (repo convention); this subsystem README is the single entry point.

## Design Policy

- One leaf package per external service, each implementing the usecase-defined `boundary.Gateway` and returning boundary output DTOs (not raw HTTP / JSON shapes)
- Each leaf wraps the `httpclient` substrate and registers a `DownstreamProfile` (a logical `Downstream` key drives the profile / breaker / metrics / budget)
- External-service profiles disable trace propagation (`PropagateTrace = false`) and reject private/loopback access (`AllowPrivateNetwork = false`) to prevent internal correlation-ID leakage and SSRF to internal hosts
- Errors are returned as `apperror` sentinels already mapped by the substrate; JSON decode / domain-shape validation failures are wrapped as `apperror.ErrUnavailable`
- The endpoint base URL is resolved at construction and injected via DI (a `NewEndpoint` per leaf); each leaf opens a span via `observability.LayerTracer` (`tf.Infra()`)

## Test Strategy

Gateways here are built on the `httpclient` substrate, not on a database, so the infrastructure layer's
real-DB strategy does not apply. Everything closes in-process behind a generated `httpclient` mock.
Leaves carry no README of their own, so these viewpoints govern every leaf under `webapi/`.

- **The boundary DTO is the assertion target, not the wire shape.** A leaf exists to stop raw JSON at
  this layer, so a test asserts the returned boundary output — including the parsed numeric / value types
  — rather than echoing back the response body it just scripted.
- **A malformed downstream response is a first-class case.** Undecodable JSON, a well-formed body whose
  shape the domain rejects, and an empty result set each get their own case, because these are the paths
  a downstream can take without any transport error to signal them.
- **The substrate's error mapping is not re-derived.** Non-2xx and transport failures arrive as
  `apperror` sentinels; assertions go through `errors.Is` against those, never a status code, since the
  status is the substrate's concern and the sentinel is what the usecase is given.
- **A cache in front of a gateway is tested on the injected clock**, never on wall time: hit, miss,
  expiry at the TTL boundary, and concurrent access to a single key. A test that sleeps to let an entry
  expire is flaky by construction.
- **Config resolution is its own subject.** Endpoint parsing rejects what it must reject at construction
  time, so a misconfigured deployment fails at startup rather than on the first outbound call.

## DI Registration

Registered by the `webapi` module in `internal/di/module/webapi.go`. Each leaf provides its constructor / endpoint and contributes its `DownstreamProfile` to the `httpclient_profiles` group.

```go
fx.Module("webapi",
    fx.Provide(
        <service>.NewEndpoint,
        <service>.New,
    ),
    provideHTTPClientProfiles(
        <service>.NewDownstreamProfile,
    ),
)
```
