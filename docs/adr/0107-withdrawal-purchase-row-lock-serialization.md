---
status: accepted
date: 2026-08-05
deciders: [maintainers]
tags: [persistence, domain, architecture, concurrency]
---

# ADR-0107: Withdrawal and purchase creation are serialized on the user row (FOR UPDATE vs FOR SHARE)

## Status

accepted

## Context

Withdrawal (`DELETE /v1/users/{userId}`, [#595]) refuses to proceed while the user still has an
in-progress purchase, and evaluates that condition inside its own transaction. Purchase creation
(`POST /v1/purchases`, [#571]) never reads `users` at all. Nothing therefore prevents this
interleaving ([#766]):

```txt
T1 (withdraw): no in-progress purchase → check passes
T2 (purchase):                            purchase created (never looks at users)
T1 (withdraw):                                                 writes deleted_at, returns 204
```

The outcome is a withdrawn user carrying an in-progress purchase. Referential integrity is intact —
the `purchases.user_id → users.id` FK holds and withdrawal is a soft delete — so only the business
invariant "a withdrawn user has no in-progress purchase" breaks.

Two properties of the codebase decide the shape of the fix:

- **Transactions run at READ COMMITTED.** `driver.NewTransactionManager` begins without `TxOptions`
  and nothing sets `default_transaction_isolation`, so PostgreSQL's default applies. A plain
  `SELECT` in the purchase transaction therefore reads its own statement snapshot and does not
  conflict with the withdrawal's uncommitted `UPDATE`. Under SERIALIZABLE this would surface as
  write skew and one side would abort with 40001, but changing the isolation level repository-wide
  is a far larger decision than this invariant warrants.
- **Withdrawal does not lock the user row.** It reads via `FindByID` and only later takes a row lock
  implicitly through `UPDATE`, which is after the in-progress check — too late to serialize anything.

## Decision

1. **Purchase creation guards the purchaser's membership under a shared row lock.** The first
   statement inside the purchase transaction is
   `SELECT u.id FROM users u WHERE u.id = $1 AND u.deleted_at IS NULL FOR SHARE`
   (`lock_active_user_share_by_id.sql`, exposed as `user.Repository.LockActiveShareByID`). Zero rows
   rejects the purchase. Shared locks are mutually compatible, so concurrent purchases by the same
   user do not serialize against each other; only the withdrawal's exclusive lock conflicts. Under
   READ COMMITTED, `EvalPlanQual` re-evaluates `deleted_at IS NULL` against the newly committed row
   version once the blocking writer commits, so a purchase that was already waiting sees the
   withdrawal and is rejected rather than proceeding on a stale snapshot.

2. **Withdrawal takes an exclusive lock on the same row, before its refusal check.**
   `FindByID` becomes `LockByID` (`lock_user_by_id.sql`, `FOR UPDATE`). The ordering is the
   load-bearing part: if the lock were taken after the in-progress check, a purchase could still
   slip in between the check and the write. The refusal conditions themselves are unchanged
   (in-progress purchase → 409, absent or already withdrawn → 404), so [#595]'s behaviour is
   preserved rather than replaced.

3. **Lock order is user row → product rows (ascending `id`), fixed across every transaction.**
   Purchase creation takes both; withdrawal takes only the user row. A single global order removes
   the possibility of a cycle, extending the ordering discipline [ADR-0100] established for product
   rows.

4. **A purchase by a withdrawn user is `ErrConflict` (409).** It mirrors withdrawal's own refusal,
   which is already 409 for the same class of collision between a user's lifecycle state and a
   requested operation. 404 is reserved for hiding the existence of *another* principal's resource
   (see `CancelPurchase`); here the subject is the caller's own state and there is nothing to hide,
   and a 404 on `POST /v1/purchases` would read as "the thing you are purchasing does not exist".
   403 belongs to the Authorizer's policy decisions, which this request has already passed.
   Errors other than not-found propagate untouched, so an outage is never reported as a withdrawal.

5. **No compensating cascade on the withdrawal side.** Because the invariant is now closed at the
   entrance, a consumer of `user.withdrawn.v1` that cancels leftover in-progress purchases would
   have nothing to act on. The existing consumer (`internal/controller/worker/withdrawalarchive`)
   keeps its archival responsibility and gains no cancellation duty.

## Consequences

### Positive Consequences

- The invariant holds under concurrency rather than probabilistically: the interleaving above is
  structurally unreachable, and the ordering is pinned by an integration test that runs two real
  transactions against the database.
- The guard costs the same number of round trips as an unlocked existence check would — one `SELECT`
  on a primary key — so strictness here is not bought with extra queries.
- Both directions of the same business collision now answer 409, and `POST /v1/purchases` already
  declared that status, so no new response code enters the API surface.

### Negative Consequences

- A purchase transaction now blocks for the duration of a concurrent withdrawal or profile update of
  the same user. Both are short, single-row transactions, but the purchase hot path is no longer
  fully independent of the user aggregate.
- `FOR SHARE` is the first shared row lock in this repository, so the lock-mode vocabulary that
  readers must hold is one wider than before.
- `internal/usecase/purchase` now depends on `user.Repository`, adding a second cross-aggregate edge
  at the usecase layer (the mirror of the existing `internal/usecase/user` → `purchase.Repository`).

### Neutral Consequences

- The window remains open for operations other than purchase creation performed with a token issued
  before withdrawal; token revocation stays rejected (see [#558]).

## Alternatives Considered

### An unlocked existence check (plain `SELECT ... WHERE deleted_at IS NULL`)

Rejected. Under READ COMMITTED it narrows the window to the width of the purchase transaction but
does not close it, because the read conflicts with nothing. A test that deterministically reproduces
the interleaving would still fail, which is the standard this invariant is held to.

### `FOR UPDATE` on the purchase side as well

Rejected as the default, though it would be correct and would reuse the existing lock precedent with
a single Repository method. Purchases by one user would then serialize against each other for no
reason: the purchase side only needs to observe that no withdrawal is in flight, and `FOR SHARE`
expresses exactly that asymmetry — shared intent to observe versus exclusive intent to change.

### `FOR KEY SHARE` on the purchase side

Rejected: it is compatible with non-key `UPDATE`s, so it would not block the withdrawal's write to
`deleted_at` and would leave the invariant unprotected.

### A database constraint (partial unique index or trigger)

Rejected. "A withdrawn user has no in-progress purchase" spans two tables and depends on the purchase
status master, so expressing it declaratively requires a trigger — which relocates business intent
into the database, away from the domain that owns it.

### SERIALIZABLE isolation

Rejected as out of scope. It would surface the interleaving as write skew and abort one side, but the
change applies to every transaction in the application and brings a retry-rate cost that this single
invariant cannot justify deciding on its own.

## Notes

- Related: [ADR-0029] (which classifies this same withdrawal check as a cross-aggregate *read* that
  needs no write atomicity — still true; this ADR adds serialization, not atomicity), [ADR-0100]
  (product-row lock ordering), [ADR-0039] (`ErrConflict` → 409), [ADR-0082] (integration tests run
  against a real database with sentinel-error rollback; the race test deviates by committing, and
  cleans up the row it creates).
- Implementation: `internal/domain/user/user_repository.go`,
  `internal/infrastructure/rdb/repository/user`, `internal/usecase/purchase`,
  `internal/usecase/user`, `database/dml/repository/user`.

[#558]: https://github.com/Tomy-ch/go-boilerplate/issues/558
[#571]: https://github.com/Tomy-ch/go-boilerplate/issues/571
[#595]: https://github.com/Tomy-ch/go-boilerplate/issues/595
[#766]: https://github.com/Tomy-ch/go-boilerplate/issues/766
[ADR-0029]: 0029-commandservice-atomicity-criterion.md
[ADR-0039]: 0039-apperror-protocol-agnostic-errors.md
[ADR-0082]: 0082-rollback-integration-tests.md
[ADR-0100]: 0100-purchase-stock-lock-and-amount-contract.md
