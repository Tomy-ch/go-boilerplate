# publisher

English | [日本語](README.ja.md)

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

## Why Abstract?

- Ensure testability of relay logic without sending real messages
- Keep the relay engine depending on a neutral port, not on `net/http` or a broker SDK
- Allow mock substitution in tests for deterministic behavior
- Decouple the outbox row representation from the transport substrate via a neutral envelope (at-least-once: a failed `Publish` is retried on the next poll)

## Implementation

`internal/infrastructure/publisher/` provides the concrete implementation that publishes messages over HTTP.
