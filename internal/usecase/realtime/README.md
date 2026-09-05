# usecase/realtime

The mechanism-side usecases of Realtime Delivery ([`docs/design/realtime-delivery.md`](../../../docs/design/realtime-delivery.md)):
what a stream connection needs decided before it is committed — whether the presented cursor can
still be replayed, and whether the presented ticket is valid — plus the issuing side of the ticket
and the read a committed connection repeats for the rest of its life.
It depends on `boundary/realtime` and `boundary/clock` only and carries no feature vocabulary.

| Usecase | Decides | Errors |
| --- | --- | --- |
| `CursorValidator.Validate(streamID, cursor)` | The replay floor, **derived** from the EventLog and its append watermark rather than stored ([ADR-0072](../../../docs/adr/0072-postgres-state-dynamodb-eventlog.md)): one strongly consistent read of the first event after `cursor`: it is `cursor+1` ⇒ replayable unless older than `realtime.EventLogRetention`; it is later than `cursor+1` ⇒ gap; nothing after ⇒ one read of `AppendedThrough`, and `cursor < watermark` ⇒ gone while `cursor >= watermark` ⇒ replayable. The first read is single, because two reads (`cursor+1`, then latest) would call a normal append that lands between them a gap | `ErrCursorExpired` (the client goes back to the canonical recovery path — History); a store failure passes through as `apperror.ErrUnavailable` so the caller can answer `503 + Retry-After` |
| `Replayer.ReadPage(streamID, after)` | What a connection has not seen yet: the events after `after`, ascending, at most `ReplayPageLimit` (64). The initial replay after a connection commits and every later re-read (a wakeup, the periodic catch-up) are the same call — only the trigger differs. Contiguity is **not** judged here: whether the returned page continues the caller's own position is the caller's comparison, since only it knows where it is | a store failure passes through as `apperror.ErrUnavailable` |
| `TicketIssuer.Issue(in)` → `TicketView` | A fresh 256-bit value from `SecretGenerator`, stored as its SHA-256 hash bound to subject / destination / scope / initial cursor, valid for `TicketTTL` (5 min) | store failures pass through |
| `TicketVerifier.Verify(value, destination)` → `realtime.StreamGrant` | The value's hash exists, is not expired at `clock.Now()`, and is bound to this destination | `ErrTicketInvalid` (wraps `apperror.ErrUnauthenticated`) for every failure — unknown, expired and wrong destination are deliberately indistinguishable: distinguishing them would tell the caller whether a given ticket exists |
| `LeaseKeeper.Beat(id)` / `Release(id)` | The instance lease ([ADR-0073](../../../docs/adr/0073-sns-sqs-instance-fanout.md)): `Beat` records "alive now" with an expiry of `clock.Now() + LeaseExpiry` (2 min); `Release` deletes the record when the instance has torn its resources down itself. The interval (`LeaseHeartbeatInterval`, 30 s) and the cleanup margin (`LeaseCleanupMargin`, 5 min) are the fixed values the heartbeat loop and the orphan-cleanup job read from here | store failures pass through |
| `OrphanSweeper.Sweep()` → `SweepResult` | Which dead instances may be reclaimed and by whom: a lease whose expiry is older than `LeaseCleanupMargin` (5 min) is claimed with a conditional write, its receiving end is reclaimed through `OrphanReclaimer`, and only then is the lease closed — conditionally again, because `Heartbeat` leaves the ownership record untouched and an instance that came back would otherwise lose a live lease. The order is fixed: the lease is the only index into the resources, so closing it before the reclaim would strand whatever the reclaim failed to delete. One failure does not stop the pass; the counts and the joined error both come back | store and reclaimer failures pass through, joined |
| `Health.Check()` / `ObserveFanout(err)` / `FanoutDegraded()` | Whether the subsystem can still deliver. The two dependencies are read differently because they answer differently: the EventLog is asked on the spot (a read of a stream nobody writes — "absent" is the healthy answer, an error is not), while the fan-out is only knowable to whoever tries to receive, so the consumer engine reports each attempt through `ObserveFanout` and the verdict is carried. One value answers all three callers — the startup probe, `/ready`, and the gate on new SSE connections — so "reachable" cannot come to mean different things at start-up and while running ([design §2.6](../../../docs/design/realtime-delivery.md)). Degradation is two-way: a fan-out that recovers clears the flag, since a one-way latch would keep refusing connections after a transient failure | `ErrFanoutUnreachable` (wraps `apperror.ErrUnavailable`) when notifications are not arriving; a store failure passes through |
| `AccessRevoker.Revoke(subject, destination)` | The revocation seam a feature calls when it withdraws a subject's access to a destination ([ADR-0074](../../../docs/adr/0074-query-ticket-stream-authentication.md)): every ticket of that pair is invalidated **first** (`StreamTicketStore.Invalidate` — a revoked ticket then fails `Verify`, so a client that ignores `STOP` cannot reconnect), and only then every serve instance is told through `RevocationNotifier` to close the matching connections. An invalidated ticket stays invalidated even when the notification fails | store and notifier failures pass through |

`ErrCursorExpired` is a package sentinel rather than an `apperror` entry: this package does not know HTTP.
The stream handler (`internal/controller/stream`) maps it to `410` by joining `apperror.ErrGone`.

Out of this package: *when* to replay (the connection registry in `internal/controller/stream` owns the
schedule and the connection state; this package only performs the read), the heartbeat *loop*
(`internal/controller/realtime` drives `LeaseKeeper.Beat` on the serve lifecycle's schedule), and
*when* to sweep (the scheduler starts `cmd job orphan-cleanup`; `internal/controller/job/orphancleanup`
only reports the counts).

## Test strategy

Boundaries are generated mocks (`boundary/realtime/mock`, `boundary/clock/mock`); no store is
opened. Each decision is pinned on both sides of its boundary — the retention edge exactly at
`EventLogRetention` and one second past it, an empty stream at the initial cursor versus a stream
whose first event is gone — and every store failure is asserted to pass through *without* being
reinterpreted as an expiry or an invalid ticket.
