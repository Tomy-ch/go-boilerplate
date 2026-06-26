# outbox

English | [日本語](README.ja.md)

Defines the `Store` persistence boundary for the transactional outbox table.
Both `emit` (usecase layer) and the relay engine (controller layer) depend on
this boundary.

```go
type Store interface {
    Insert(ctx context.Context, p EmitParams) (uuid.UUID, error)
    ClaimPending(ctx context.Context, limit int32) ([]PendingMessage, error)
    MarkPublished(ctx context.Context, id int64) error
    MarkFailed(ctx context.Context, id int64, lastErr string) (attempts int32, err error)
    MarkDead(ctx context.Context, id int64) error
    ReplayDead(ctx context.Context, messageID *uuid.UUID) (int64, error)
    DeletePublished(ctx context.Context, cutoff time.Time, limit int32) (int64, error)
    OldestPendingCreatedAt(ctx context.Context) (createdAt time.Time, ok bool, err error)
}
```

## Why Abstract?

- Ensure testability of emit / relay / GC logic without a real database
- Keep Usecase and the relay engine depending on a persistence port, not on sqlc or `FOR UPDATE SKIP LOCKED` SQL details
- Allow mock substitution in tests for deterministic behavior
- Share one contract between the in-transaction `Insert` (usecase) and the claim/mark/replay/SLI calls (relay engine)

## Implementation

`internal/infrastructure/rdb/system_query/outbox/` provides the concrete RDB implementation backed by sqlc-generated queries.
