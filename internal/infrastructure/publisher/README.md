# publisher

English | [日本語](README.ja.md)

`internal/infrastructure/publisher` is **the only place that chooses the implementation of the transactional outbox publish boundary** (`publisher.Publisher`). It ships the HTTP implementation, which POSTs outbox messages claimed by the relay engine to a receiving endpoint, and selects between implementations by the `OUTBOX_PUBLISHER` discriminator.

## Architectural Position

```mermaid
flowchart TB
    subgraph "Usecase Layer"
        IF["publisher.Publisher interface"]
    end
    subgraph "Infrastructure Layer"
        Impl["httpPublisher impl"]
        Sub["httpclient.Client substrate"]
    end

    Impl -. implements .-> IF
    Impl --> Sub
```

Implements the `publisher.Publisher` interface (`internal/usecase/boundary/publisher`) in the Infrastructure layer, delegating actual transport to the `httpclient` substrate. The relay engine and usecase depend only on the boundary, not on HTTP details.

## Choosing an implementation

`New(cfg, client, tf)` switches on `OUTBOX_PUBLISHER` and returns the matching adapter; an unknown value fails startup rather than falling through to a default, so a typo never publishes to an unintended target. The publish target is a per-deployment decision rather than a function of the environment tier, which is why it is an explicit discriminator instead of an `APP_ENV` branch.

Each branch resolves its own settings, so a deployment that publishes to a queue is never asked for `OUTBOX_ENDPOINT`, and vice versa. Both resolutions fail at relay startup rather than at the first publish — an unset target would otherwise dead-letter every message silently.

<!-- sample-api:begin -->
The `sqs` branch — the only branch besides `http` — is wiring from the removable sample set (see [ADR-0048](../../../docs/adr/0048-broker-sdk-isolation-measured-as-coupling.md)); after `make setup-remove-sample-api` only the HTTP branch remains, while the SQS adapter itself stays as an unwired reference implementation.
<!-- sample-api:end -->

## Design Policy

- Transport retry is disabled (`MaxAttempts = 1`): the relay poll loop is itself the at-least-once retry body, so substrate-level retry would double up (D10). Redelivery is owned by the next relay poll.
- The non-idempotent POST carries `MessageID` as `Idempotency-Key` for receiver-side dedup, but `AllowRetry` is explicitly `false`.
- Trace propagation is disabled (`PropagateTrace = false`): the `traceparent` captured at emit time is propagated explicitly via message headers, so the substrate's automatic injection is suppressed.
- The endpoint URL is resolved once from config and injected at construction; `Content-Type: application/json` plus the message's own headers (e.g. `traceparent`) are sent.
- Non-2xx / transport failures are mapped to `apperror` sentinels by the substrate and returned as-is, signaling the relay to retry on the next poll.

## Test Strategy

The substrate here is the `httpclient.Client` boundary, not a database, so the infrastructure layer's
real-DB strategy does not apply. Everything closes in-process: the downstream is a generated
`httpclient` mock, and nothing is sent over a network.

- **The request is asserted, not just the outcome.** The adapter's whole job is to turn an outbox
  message into one HTTP call, so the test inspects the `Request` handed to the substrate — method,
  endpoint, `Content-Type`, the message's own headers (`traceparent`), and `MessageID` carried as
  `Idempotency-Key`. Checking only the returned error would leave the mapping free to drift.
- **The disabled knobs are pinned as deliberately off**, because they are safeguards rather than
  defaults: `AllowRetry = false` (the relay poll loop owns redelivery) and `PropagateTrace = false`
  (the emit-time `traceparent` travels as a message header instead). Both would fail silently if
  flipped, which is exactly why each gets its own case.
- **Sensitive headers are pinned against normalisation gaps.** Header matching must not be defeated by
  case or surrounding whitespace, so those forms are tested explicitly rather than assumed.
- **Substrate errors propagate unchanged.** A non-2xx or transport failure is already an `apperror`
  sentinel when it arrives; the assertion is `errors.Is` against that sentinel, confirming the adapter
  neither re-wraps nor flattens it — the relay's retry decision depends on it surviving intact.
- **Implementation selection is its own subject.** `New` switching on `OUTBOX_PUBLISHER` is tested for
  each known value *and* for an unknown one, since failing startup on a typo is the contract; so is each
  branch resolving only its own settings, so a queue deployment is never asked for `OUTBOX_ENDPOINT`.

## DI Registration

Registered by the `outbox_publisher` module in `internal/di/module/outboxpublisher.go`. The downstream profile is contributed to the `httpclient_profiles` group.

```go
fx.Module("outbox_publisher",
    fx.Provide(
        outboxpublisher.New,
    ),
    provideHTTPClientProfiles(
        outboxpublisher.NewDownstreamProfile,
    ),
)
```
