# usecase/realtime

The mechanism-side usecases of Realtime Delivery ([`docs/design/realtime-delivery.md`](../../../docs/design/realtime-delivery.md)):
what a stream connection needs decided before it is committed — whether the presented cursor can
still be replayed, and whether the presented ticket is valid — plus the issuing side of the ticket.
It depends on `boundary/realtime` and `boundary/clock` only and carries no feature vocabulary.

| Usecase | Decides | Errors |
| --- | --- | --- |
| `CursorValidator.Validate(streamID, cursor)` | The replay floor, **derived** from the EventLog rather than stored ([ADR-0072](../../../docs/adr/0072-postgres-state-dynamodb-eventlog.md)): one strongly consistent read of the first event after `cursor`: it is `cursor+1` ⇒ replayable unless older than `realtime.EventLogRetention`; it is later than `cursor+1` ⇒ gap; nothing after and the event at a non-initial cursor absent ⇒ gone. A single read, because two reads (`cursor+1`, then latest) would call a normal append that lands between them a gap | `ErrCursorExpired` (the client goes back to the canonical recovery path — History); a store failure passes through as `apperror.ErrUnavailable` so the caller can answer `503 + Retry-After` |
| `TicketIssuer.Issue(in)` → `TicketView` | A fresh 256-bit value from `SecretGenerator`, stored as its SHA-256 hash bound to subject / destination / scope / initial cursor, valid for `TicketTTL` (5 min) | store failures pass through |
| `TicketVerifier.Verify(value, destination)` → `VerifiedTicketView` | The value's hash exists, is not expired at `clock.Now()`, and is bound to this destination | `ErrTicketInvalid` (wraps `apperror.ErrUnauthenticated`) for every failure — unknown, expired and wrong destination are deliberately indistinguishable |

`ErrCursorExpired` is a package sentinel rather than an `apperror` entry: the taxonomy has no `410`
and nothing outside this package maps it yet; the stream handler (Phase 6) owns that mapping.

Out of this package: replay reads and catch-up (the stream handler owns them together with the
connection state), lease heartbeat (serve lifecycle), orphan-cleanup ownership (the cleanup job),
and the revocation seam (Phase 5).

## Test strategy

Boundaries are generated mocks (`boundary/realtime/mock`, `boundary/clock/mock`); no store is
opened. Each decision is pinned on both sides of its boundary — the retention edge exactly at
`EventLogRetention` and one second past it, an empty stream at the initial cursor versus a stream
whose first event is gone — and every store failure is asserted to pass through *without* being
reinterpreted as an expiry or an invalid ticket.
