# outbox

Defines the `Store` persistence boundary for the transactional outbox.
Both `emit` (usecase layer) and the relay engine (controller layer) depend on
this boundary.

```go
type Store interface {
    Insert(ctx context.Context, p EmitParams) (uuid.UUID, error)
    ClaimPending(ctx context.Context, channel Channel, limit int32) ([]PendingMessage, error)
    MarkPublished(ctx context.Context, id int64) error
    MarkFailed(ctx context.Context, id int64, lastErr string, nextAttemptAt time.Time) error
    MarkDead(ctx context.Context, id int64) error
    ReplayDead(ctx context.Context, messageID *uuid.UUID) (int64, error)
    DeletePublished(ctx context.Context, cutoff time.Time, limit int32) (int64, error)
    OldestPendingCreatedAt(ctx context.Context, channel Channel) (createdAt time.Time, ok bool, err error)
    CountBlockedStreams(ctx context.Context, channel Channel) (int64, error)
}
```

A row belongs to exactly one **delivery channel** (`Channel`), and a relay process
claims exactly one channel, so a stalled channel never holds up another. A row may
additionally carry an **ordering key** and a position within it; `ClaimPending`
refuses a row while an earlier position on the same key is still unpublished, which
is what makes a stream's delivery a contiguous prefix rather than a set with holes.

`MarkFailed` does not dead-letter. A row becomes `dead` only through `MarkDead`,
which the relay calls when the publisher classified the failure as permanent
(see [ADR-0058](../../../../docs/adr/0058-outbox-dead-on-permanent-error.md)).

## Why Abstract?

- Ensure testability of emit / relay / GC logic without a real database
- Keep Usecase and the relay engine depending on a persistence port, not on sqlc or `FOR UPDATE SKIP LOCKED` SQL details
- Allow mock substitution in tests for deterministic behavior
- Share one contract between the in-transaction `Insert` (usecase) and the claim/mark/replay/SLI calls (relay engine)

## Implementation

`internal/infrastructure/rdb/system_cqrs/outbox/` provides the concrete RDB implementation backed by sqlc-generated queries.
