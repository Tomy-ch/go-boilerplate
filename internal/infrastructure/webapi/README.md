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

> **Departure from Evans.** Structurally this is an Anticorruption Layer: a port stated in our own
> vocabulary, translation at the boundary, nothing of the vendor reaching inward. The stated motive is
> not Evans's, though. Everything above argues from dependency inversion, substitutability, and
> transport hiding — Evans argues from semantics, from keeping the upstream *model* out. The practical
> difference is what the layer reliably protects. Types and vendor vocabulary: yes, by construction.
> Concepts: not decided here. Nothing above says which side wins when an external service's notion of
> a thing genuinely disagrees with ours. Until it does, resolve such a conflict toward this model's
> vocabulary and record the choice where the leaf translates.

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
