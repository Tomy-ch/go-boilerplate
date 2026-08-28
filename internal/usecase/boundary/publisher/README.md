# publisher

Defines the outbound `Publisher` boundary for domain events and a
substrate-agnostic message envelope. Both the relay engine (controller layer)
and the publish adapter (infrastructure layer) depend on this boundary.

```go
type Publisher interface {
    Publish(ctx context.Context, m Message) error
}

type Message struct {
    MessageID uuid.UUID
    EventType string
    Payload   []byte
    Headers   map[string]string
}
```

An implementation may classify a failure it returns with `apperror.ErrPermanent`
(a retry cannot change the outcome) or `apperror.ErrRetryable` (it might). The relay
dead-letters an entry on the former and keeps retrying on the latter; an error carrying
neither classification is treated as retryable, so a publisher that cannot tell never
costs a message (see [ADR-0058](../../../../docs/adr/0058-outbox-dead-on-permanent-error.md)).

## Why Abstract?

- Ensure testability of relay logic without sending real messages
- Keep the relay engine depending on a neutral port, not on `net/http` or a broker SDK
- Allow mock substitution in tests for deterministic behavior
- Decouple the outbox row representation from the transport substrate via a neutral envelope (at-least-once: a failed `Publish` is retried on the next poll)

## Implementation

`internal/infrastructure/publisher/` provides the concrete implementation that publishes messages over HTTP.
