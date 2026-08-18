---
status: accepted
date: 2026-08-05
deciders: [maintainers]
tags: [persistence, domain, architecture, concurrency]
---

# ADR-0035: Serialize contended writes with ordered pessimistic row locks taken before the guarded condition

## Status

accepted

## Context

A write that reads a row, decides something from what it read, and then writes, is serialized
against a concurrent write by nothing the application does by default. Transactions here run at
**READ COMMITTED** — the transaction manager begins without `TxOptions` and nothing sets
`default_transaction_isolation`, so PostgreSQL's default applies — and a plain `SELECT` reads its
own statement snapshot, conflicting with nothing. Any invariant of the form *"this write is
allowed only while that other row is in state S"* therefore carries a window between the check
and the write:

```txt
T1: reads the guard row → condition holds → proceeds
T2:                       changes the guard row, commits
T1:                                          writes anyway
```

Referential integrity survives this interleaving, so no database constraint reports it; only the
business invariant breaks. Under SERIALIZABLE the same interleaving would surface as write skew
and one side would abort with `40001`, but the isolation level is a property of *every*
transaction in the application, and raising it repository-wide — with the retry-rate cost that
carries — is a far larger decision than any single invariant can justify on its own.

The serialization must therefore be expressed per invariant, with pessimistic row locks. That
leaves four questions to settle: in what order locks are taken, at what point in the transaction,
in which mode, and who is allowed to author the condition the lock protects. A fifth follows from
the answers: what the application returns when the guarded condition turns out to be false.

## Decision

1. **Locks are taken in a single global order, fixed across every transaction.** Rows are locked
   by a stable key in ascending order (`... WHERE id = ANY($1) ORDER BY id FOR UPDATE`), and where
   a workflow locks rows in more than one table, the table order is fixed too — a workflow that
   locks a subset takes those rows in the same relative order as one that locks all of them. A
   cycle in the wait-for graph requires two transactions to acquire the same pair in opposite
   orders; a single global order makes that unreachable, so deadlock is removed **structurally**
   rather than mitigated by retry. A request carrying a duplicate key is rejected as a validation
   error before locking, so the ordering premise holds.

2. **The lock is taken before the condition it protects is evaluated.** This is the load-bearing
   part, and the part that is easy to get wrong: a transaction that evaluates its refusal
   condition first and takes the row lock only later — often implicitly, through its own `UPDATE`
   — has already left the window open. A lock acquired after the decision it was meant to guard
   serializes nothing.

3. **Lock mode expresses the asymmetry between observing and changing.** A reader whose
   observation must not be invalidated takes a **shared** lock on the guard row; the writer that
   would invalidate it takes an **exclusive** lock on the same row. Shared locks are mutually
   compatible, so concurrent readers of the same guard row do not serialize against one another —
   only the invalidating writer conflicts. Taking the exclusive lock on both sides would also be
   correct and would need one concept fewer, but it serializes observers against each other for no
   reason: the observing side only needs to establish that no invalidating write is in flight, and
   the shared lock states exactly that. A *key-share* lock is not sufficient — it is compatible
   with non-key `UPDATE`s, so it fails to block precisely the write it exists to observe.

4. **A waiter re-evaluates; it does not resume on its entry snapshot.** Under READ COMMITTED, when
   a blocking writer commits, PostgreSQL's `EvalPlanQual` re-evaluates the blocked statement's
   predicate against the newly committed row version. A transaction that was already waiting
   therefore observes the change — its lock query returns zero rows — instead of proceeding on the
   snapshot it entered with. This is what makes a locking `SELECT` a usable guard rather than
   merely a wait.

5. **The lock query acquires the row and returns state; it does not author the criterion.** A
   locking `SELECT` narrows to the row it must lock and returns that row, but the *business
   condition* the lock protects is defined by a domain predicate over the returned state, never by
   the SQL's `WHERE`. A lock query that also filters on the guarded condition looks economical and
   is the wrong shape: it relocates a business rule into infrastructure, where the domain no longer
   owns it and a second copy exists to diverge silently the first time only one of them moves —
   and it collapses "the row is absent" and "the row is present but ineligible" into one
   indistinguishable zero-row result. This is the criterion-authorship rule already recorded in
   [`internal/domain/README.md`](../../internal/domain/README.md) (§ Query and Aggregate) and
   [`docs/rules.md`](../rules.md) (§ Domain Layer Constraints); a locking read is bound by it like
   any other read. Restating the guard inside the write statement itself is permitted only as a
   fail-closed second net **derived from** the domain rule
   ([ADR-0031](0031-lightweight-cqrs.md) § Derivation), never as its author.

   The resulting division of labour has three parts. The **usecase** takes the lock, owns the
   transaction boundary, and maps a refused condition onto a protocol error. The **domain
   predicate** on the locked aggregate decides whether the condition holds. Where the rule spans
   aggregates — the locked row belongs to one aggregate and the evidence against it to another —
   the rule lives in a **Domain Service** (`internal/domain/service/<name>/`), which is the one
   place permitted to import more than one aggregate and which returns the domain error the
   usecase maps. The locking Repository method therefore hands back an aggregate rather than a
   `bool`: a `bool` would mean infrastructure had already decided, which is precisely what this
   point forbids.

6. **A collision between the caller's own lifecycle state and the requested operation is a
   conflict (409).** When the guarded condition is false because of the state of the principal
   making the request, the answer is `ErrConflict` → 409
   ([ADR-0046](0046-apperror-protocol-agnostic-errors.md)). 404 is reserved for hiding the
   existence of *another* principal's resource; here the subject is the caller's own state, there
   is nothing to hide, and a 404 on a creation endpoint would read as "the thing you are creating
   against does not exist". 403 belongs to the authorizer's policy decisions, which the request has
   already passed. Both directions of a mutually exclusive pair of operations therefore answer the
   same status, so the pair reads as one rule rather than two unrelated refusals. Errors other than
   not-found propagate untouched, so an infrastructure outage is never reported as a business
   refusal.

## Consequences

### Positive Consequences

- The invariant holds under concurrency rather than probabilistically: the interleaving above
  becomes structurally unreachable, and the ordering is pinned by an integration test that runs two
  real transactions against the database ([ADR-0091](0091-rollback-integration-tests.md)).
- Deadlock is avoided by construction rather than absorbed by the transaction retry
  ([ADR-0034](0034-transaction-retry-idempotent-callers.md)), so the retry budget stays available
  for genuine serialization failures.
- A guard costs the same number of round trips as an unlocked existence check would — one `SELECT`
  on a primary key — so strictness here is not bought with extra queries.
- Both directions of one business collision answer 409, so closing an invariant introduces no new
  status code into the API surface.

### Negative Consequences

- Pessimistic locks serialize concurrent operations on the same row. This is acceptable where the
  row is a genuinely contended resource, but it bounds write throughput per hot row.
- A transaction that guards on another aggregate's row now blocks for the duration of any
  concurrent writer of that row, so a hot path is no longer fully independent of the aggregate it
  guards on.
- Two lock modes widen the vocabulary a reader must hold: seeing a locking read is no longer enough,
  the mode has to be read too.
- A workflow that locks another aggregate's row acquires a dependency on that aggregate. Confining
  the locking reads to their own narrow repository interface limits the dependency to the methods
  actually used, at the cost of that aggregate presenting more than one repository interface.

### Neutral Consequences

- A row lock serializes the transactions that take it and says nothing about authority obtained
  before a transaction started; a credential issued before the guard row changed is a separate
  concern with a separate mechanism.

## Alternatives Considered

### An unlocked existence check (plain `SELECT ... WHERE <condition>`)

Rejected. Under READ COMMITTED it narrows the window to the width of the guarded transaction but
does not close it, because the read conflicts with nothing. A test that deterministically
reproduces the interleaving still fails — and that is the standard an invariant is held to here.

### Advisory locks (`pg_advisory_xact_lock`)

Rejected. Advisory locks fit logical keys that have no backing row. Where the contended resource
*is* a row, the row lock is the direct expression of it and needs no separate key convention. The
outbox relay's `SKIP LOCKED` choice ([ADR-0055](0055-skip-locked-outbox-relay.md)) is a different
contention profile — many workers competing to claim *any* queue row — and does not transfer to a
specific row that every contender must observe.

### Optimistic locking (a version column plus retry)

Rejected for this class of invariant. Optimistic control detects a conflict only on rows the
transaction itself writes; a guard row that a transaction merely *reads* carries no version bump,
so exactly the write skew at issue stays invisible to it. It would also convert a bounded wait into
a retry loop whose cost grows with contention.

### The exclusive lock on both sides

Rejected as the default, though it is correct and would reuse a single lock mode and a single
repository method. Operations that only need to *observe* the guard row would then serialize against
each other for no reason; the shared/exclusive pair expresses the real asymmetry — shared intent to
observe versus exclusive intent to change.

### A database constraint (partial unique index or trigger)

Rejected. An invariant that spans two tables and depends on a reference master cannot be expressed
declaratively; it requires a trigger, which relocates business intent into the database and away
from the domain that owns it — the same objection as point 5, in a stronger form.

### SERIALIZABLE isolation

Rejected as out of scope. It would surface the interleaving as write skew and abort one side, but
it applies to every transaction in the application and brings a retry-rate cost that a single
invariant cannot justify deciding on its own.

## Notes

- Related: [ADR-0031](0031-lightweight-cqrs.md) (Repository vs CommandService, and the Derivation
  rule that binds a restated guard), [ADR-0033](0033-commandservice-atomicity-criterion.md) (the
  three-way procedure that decides *whether* a cross-aggregate condition must be held — its branch 2
  is the entry point into this ADR — and when a step needs write atomicity rather than only
  serialization),
  [ADR-0034](0034-transaction-retry-idempotent-callers.md) (serialization-failure retry),
  [ADR-0046](0046-apperror-protocol-agnostic-errors.md) (`ErrConflict` → 409),
  [ADR-0055](0055-skip-locked-outbox-relay.md) (the contrasting claim-a-queue-row profile),
  [ADR-0091](0091-rollback-integration-tests.md) (integration tests against a real database).
- Which rows a given workflow locks, and which business rule each lock protects, is feature content
  rather than an architectural decision, so it is specified with the feature. In this repository
  that means the removable sample set (`docs/spec/purchase/`, `docs/spec/user/`) — referenced by
  path rather than linked, because those files are deleted by `make setup-remove-sample-api` while
  this ADR stays.
