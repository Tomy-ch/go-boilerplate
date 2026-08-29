# realtime

The seam of Realtime Delivery ([`docs/design/realtime-delivery.md`](../../../../docs/design/realtime-delivery.md)):
the feature-neutral envelope `DeliveryEvent` and the boundaries implemented by whoever stores and
delivers it. Usecase code (`internal/usecase/realtime/`, the feature's realtime adapter) depends on
this package only; the DynamoDB stores, the PostgreSQL sequence allocator and the SNS / SQS fan-out
adapter (infrastructure) implement it. Neither vendor vocabulary (table / partition / TTL) nor feature vocabulary
(conversation / message / operator) appears here.

```go
// The envelope. EventID equals the outbox message_id and decides idempotency.
type DeliveryEvent struct {
    EventID       string
    StreamID      StreamID
    Sequence      Sequence        // int64; String() is the decimal wire form
    Type          string
    OccurredAt    time.Time
    SchemaVersion int
    Payload       json.RawMessage
}

func (e DeliveryEvent) Validate() error          // ErrInvalidEvent / ErrPayloadTooLarge (64 KiB serialized)
func (e DeliveryEvent) MarshalJSON() ([]byte, error)

type EventLogStore interface {
    Append(ctx context.Context, event DeliveryEvent) error                        // idempotent on EventID; ErrSequenceConflict otherwise
    ReadAfter(ctx context.Context, q ReadAfterQuery) (ReadAfterResult, error)     // strongly consistent, ascending, HasMore
    Latest(ctx context.Context, streamID StreamID) (DeliveryEvent, bool, error)
    Find(ctx context.Context, streamID StreamID, seq Sequence) (DeliveryEvent, bool, error)
}

type StreamTicketStore interface {
    Save(ctx context.Context, ticket StreamTicket) error
    Find(ctx context.Context, hash TicketHash, asOf time.Time) (StreamTicket, bool, error)  // expired ⇒ ok=false
    Invalidate(ctx context.Context, subject string, destination StreamID) error
}

type InstanceLeaseStore interface {
    Heartbeat(ctx context.Context, lease InstanceLease) error
    Delete(ctx context.Context, id InstanceID) error
    ListExpired(ctx context.Context, asOf time.Time) ([]InstanceLease, error)
    AcquireCleanup(ctx context.Context, claim CleanupClaim) (bool, error)         // conditional; false = someone else owns it
}

type SecretGenerator interface {
    Generate() (string, error)   // 256-bit opaque ticket value
}

// The instance's own queue on the fan-out (one per serve instance, created at start, deleted at stop).
type InstanceSubscription interface {
    Provision(ctx context.Context, id InstanceID) error            // idempotent per id
    Receive(ctx context.Context, limit int) ([]Notification, error) // bounded wait; returns on ctx
    Delete(ctx context.Context, n Notification) error               // undeleted notifications are redelivered
    Teardown(ctx context.Context) error                             // unregister, then delete; no-op if never provisioned
}

type Notification struct {            // Kind selects which of the two is populated
    Kind       NotificationKind      // KindWakeup | KindRevocation | KindUnknown ("": unreadable; delete it)
    Wakeup     Wakeup                // EventID / StreamID / Sequence — "re-read the stream after your cursor"
    Revocation Revocation            // Subject / Destination — "close that subject's connections"
    Receipt    string                // substrate-specific delete key; opaque here
}
```

## Invariants the boundary carries

| Invariant | Where it is enforced |
| --- | --- |
| A serialized event is at most `MaxSerializedBytes` (64 KiB) — the whole envelope, not the payload alone | `DeliveryEvent.Validate`, called by the emitting adapter before the outbox row is written and by every `EventLogStore.Append` before it stores |
| `Sequence` is decimal on the wire, gap-free in a stream, and its zero value carries no meaning — "not yet allocated" is the `ok` of `SequenceAllocator.Current`, never a sentinel value | `Sequence.String`; the allocator and ADR-0072 |
| Re-appending the same `EventID` at the same position succeeds; a different `EventID` there is `ErrSequenceConflict` | `EventLogStore.Append` (the outbox relay retries without a special case) |
| A cursor is only meaningful for the ticket's own `Destination` | `StreamTicket.Destination`; the stream handler compares them |
| Expiry is decided by the caller's clock (`asOf`), never by the store's clean-up | `StreamTicketStore.Find`, `InstanceLeaseStore.ListExpired` / `AcquireCleanup` |

`EventLogRetention` (7 days) is defined here because both sides read it — the store to expire items, the
usecase to derive the replay floor. Ticket TTL and lease heartbeat / expiry / cleanup margin are owned by
`internal/usecase/realtime/`; the stores receive the resulting timestamps and do not know the numbers.

## Why a separate `SecretGenerator`

`boundary/token` exists for the cart's session tracking and is removed together with the sample
feature (`scripts/setup/remove-sample-api/sample-manifest.ts`). Realtime Delivery must compile and test
after that removal, so it carries its own randomness seam; the implementation lives in
`internal/infrastructure/realtimesecret/`.

## Implementation

| Boundary | Implementation |
| --- | --- |
| `EventLogStore` | `internal/infrastructure/eventlog/dynamodb/` |
| `StreamTicketStore` | `internal/infrastructure/streamticket/dynamodb/` |
| `InstanceLeaseStore` | `internal/infrastructure/instancelease/dynamodb/` |
| `SecretGenerator` | `internal/infrastructure/realtimesecret/` |
| `SequenceAllocator` (`sequence.go`) | `internal/infrastructure/rdb/system_cqrs/realtime/` |
| `RevocationNotifier` (`revocation.go`) | `internal/infrastructure/realtime/aws/` (publishes the revocation on the same topic as the wakeups); called by `usecase/realtime.AccessRevoker` after the tickets are invalidated |
| `InstanceSubscription` (`fanout.go`) | `internal/infrastructure/realtime/aws/` (the instance's SQS queue and its SNS subscription); driven by the consumer engine in `internal/controller/realtime/` |

Mocks are generated per file into `mock/` (`go generate ./...`).
