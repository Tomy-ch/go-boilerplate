---
status: accepted
date: 2026-08-05
deciders: [maintainers]
tags: [persistence, cqrs, architecture, concurrency]
---

# ADR-0031: Reserve CommandService for multi-aggregate writes that require single-transaction atomicity

## Status

accepted

## Context

[ADR-0029](0029-lightweight-cqrs.md) defines the *structure* of the CommandService
construct: its interface lives in the usecase layer (`internal/usecase/<workflow>/command/`),
its implementation lives under `internal/infrastructure/rdb/command_service/<aggregate>/`, it
returns DTOs, and it is registered in `persistenceModule` and injected via DI. What ADR-0029
deliberately leaves open is the *placement criterion*: when does a write operation actually
warrant a CommandService instead of a regular usecase composed of Repository calls?

Without such a criterion, the placement of multi-aggregate write operations falls back to
implementer judgment about inter-aggregate semantics, and decisions vary by implementer. The
naive heuristic — "it touches more than one aggregate, so it needs a CommandService" — pulls
work that could be eventually consistent into synchronous transactions, contradicting this
project's outbox-first design ([ADR-0051](0051-transactional-outbox.md)). A related ambiguity
exists in ADR-0029 itself: its consequences note that CommandService "can freely optimize
flexible updates, deletes", which could be misread as write-shape flexibility or performance
being sufficient grounds for introducing one.

That question is about *writes*, and it does not reach every way an operation crosses an aggregate
boundary. An operation may also **read** another aggregate to decide whether it is permitted at all
— a **guard**. A guard writes nothing, so an atomicity criterion never triggers on it and the
operation lands on the default path by omission rather than by decision. Under READ COMMITTED,
however, a condition that was merely read is not held: a concurrent write can invalidate it between
the check and the commit ([ADR-0033](0033-ordered-pessimistic-row-locks.md)). "It spans aggregates
but it is a read, so no write atomicity is required" is therefore a true sentence with a right
conclusion — a regular usecase — and an incomplete answer, because the read may still have to be
serialized against the write that would invalidate it. Left without a branch of its own, the choice
between holding a cross-aggregate condition synchronously and letting it go stale falls back to
exactly the implementer judgment this ADR exists to remove.

The goal is to decide where an operation that crosses an aggregate boundary belongs — for its
writes and for the conditions it reads — based on a verifiable criterion rather than implementer
judgment.

## Decision

**CommandService is defined as the write-side counterpart of QueryService.** Here,
CommandService is not a layer; it is a processing category contrasted with QueryService.

### Definition

QueryService and CommandService are defined as the same kind of construct:

- **QueryService** — the residue that remains when read performance requirements forbid
  decomposition into per-aggregate reads.
- **CommandService** — the residue that remains when write atomicity (immediacy) requirements
  forbid decomposition into eventual consistency.

Neither is a semantic domain category. **Both are constructs that stand where non-functional
requirements forbid aggregate decomposition.**

The underlying fact: given unlimited latency and resources, every read decomposes into
per-aggregate Repository reads joined in application code, and every write decomposes into a
single-aggregate write followed by an eventually consistent cascade. QueryService and
CommandService are needed only where non-functional requirements forbid that decomposition.

### Criterion

Two independent questions decide where a cross-aggregate operation belongs. They are independent
because one is about reading and the other about writing, and a single operation may answer yes to
both:

1. **Does a condition read from another aggregate have to hold for the rest of the transaction?**
   Equivalently: can a concurrent operation invalidate the checked condition between the check and
   the commit? This asks whether a *read may go stale*, not whether a write is atomic, which is why
   the second question never reaches it.
2. **Does the multi-aggregate write require single-transaction atomicity?** Immediacy — all effects
   being visible at API response time — is the typical reason this requirement arises.

### Decision procedure

When an operation crosses an aggregate boundary:

1. **Decomposition (default).** Ask the requirements two things: whether the consequence for the
   other aggregate may be eventually consistent (a cascade via outbox events,
   [ADR-0051](0051-transactional-outbox.md)), and whether a condition read from another aggregate
   may go stale after it has been checked. When both answers are yes, implement it as a regular
   usecase, propagating any consequence as an outbox event. The condition is read without a lock and
   no other aggregate is held inside the transaction.
2. **Guard (synchronous row lock; still a regular usecase).** Does a **guard** need to hold for the
   duration of the transaction — i.e. can a concurrent operation invalidate the checked condition
   between the check and the commit? If so, a synchronous row lock is required, and the operation
   stays a normal usecase. Take the lock before evaluating the condition and in the global lock
   order ([ADR-0033](0033-ordered-pessimistic-row-locks.md)); where the rule spans aggregates it
   lives in a Domain Service. This branch buys immediate consistency for a *read*. It does not make
   any write atomic, and is never on its own a reason to introduce a CommandService.
3. **Atomicity (CommandService; exception, must be justified).** Only when single-transaction
   atomicity of the multi-aggregate *write* remains as a requirement.

The default is decomposition; CommandService is a justified exception. Two justifications are not
acceptable: "it spans multiple aggregates, therefore CommandService", and — the failure mode branch
2 exists to close — "it is only a read, therefore nothing is needed".

### Departure from "1 Aggregate = 1 Transaction Boundary"

Branches 2 and 3 both put rows belonging to more than one aggregate inside a single transaction, so
both depart from the principle [`internal/domain/README.md`](../../internal/domain/README.md)
(§ Aggregate Boundary) states as "1 Aggregate = 1 Transaction Boundary" — Vernon's formulation of
Evans's argument that the aggregate is the unit of consistency. The departure is stated here rather
than left implicit:

- **What Evans holds.** The aggregate is the boundary of *immediate* consistency. One transaction
  changes one aggregate; anything beyond it is reconciled afterwards. A boundary that may be widened
  whenever it is convenient constrains nothing, so the discipline is the point of the pattern.
- **What this repository does instead.** Two named widenings, and only these two. A guard locks a row
  belonging to another aggregate and holds it until commit (branch 2). A write that must be atomic
  across aggregates runs in one transaction through a CommandService (branch 3).
- **Why.** Evans's argument is about *change*: the hazard in a wide boundary is mutating several
  aggregates through one loaded graph until no one can say which invariant belongs to which root.
  Branch 2 does not mutate the other aggregate at all — it observes one row and blocks the writer
  that would invalidate the observation, so that aggregate's root keeps sole authority over its own
  changes. Branch 3 does mutate more than one, which is why it is the narrow exception: it is
  admitted only where the requirements say the intermediate state must never be observable, and the
  condition it enforces is still authored by the domain
  ([ADR-0029](0029-lightweight-cqrs.md) § Derivation). What both branches refuse is the option a
  strict reading would otherwise leave as the default — deciding a cross-aggregate question from a
  read that nothing holds, which is not eventual consistency but no consistency at all.

### Worked instances

- **Purchase creation — branch 2 and branch 3 together.** Stock validation and decrement must be
  atomic with purchase confirmation: a state where "a purchase succeeded without stock" must never
  be observable, even momentarily (no overselling). The writes across the aggregates involved
  <!-- 撤去後にこの箇所へ自分の例を置くための指針。
       目的: どの表へ同時に書くのかが見えないと、原子性の範囲が伝わらない。
       意義: 効くのは「途中状態が観測されてはならない」ことで、表の数や名前ではない。
       書き方: 同一トランザクションで書かれる表を列挙する。 -->
  <!-- sample-api:begin -->
  （サンプルでは `purchases` / `purchase_details` / `products`）
  <!-- sample-api:end -->
  therefore require single-transaction atomicity (branch 3), and the outbox insert joins
  that same transaction as usual per [ADR-0051](0051-transactional-outbox.md). The outbox insert is
  not what justifies the CommandService — a regular usecase also writes the outbox in its own
  transaction; the justification is the stock/purchase atomicity. Independently, the same
  transaction guards on the purchaser still being a member, and that condition can be invalidated by
  a concurrent withdrawal, so the user row is locked first (branch 2). The transaction consequently
  spans three aggregates — user, product, purchase — for two different reasons, which is why the two
  branches are asked separately. Specified in `docs/spec/purchase/usecase.md`.
- **User withdrawal — branch 2 for the guard, branch 1 for the cascade.** The core of withdrawal is
  a single-aggregate write to `users.deleted_at`. The cascade — cancelling pending purchases and
  restoring stock — requires no immediacy and is eventually consistent via outbox events (branch 1).
  The check "cannot withdraw with purchases in progress" spans aggregates and is a read, so branch 3
  does not apply and the operation stays a usecase; but a concurrent purchase creation would
  invalidate it, so branch 2 does apply and the user row is locked exclusively before the check.
  This is the shape the two-way procedure could not describe: a usecase that nonetheless takes a
  lock. Specified in `docs/spec/user/usecase.md`.
- **Product unpublication — branch 1, with no guard at all.** Clearing a product's publication date
  is a single-aggregate write, and the cross-aggregate condition it might appear to threaten — the
  in-progress purchases that reference the product — is deliberately not guarded. A purchase records
  a unit-price snapshot when it is created, and no step of its lifecycle re-reads the product's
  publication state, so the product going unpublished cannot make the purchase wrong. The condition
  is therefore allowed to go stale, and the update takes no lock on any purchase row. This is the
  contrasting instance to the withdrawal guard: same shape (one aggregate's write, another
  aggregate's state), opposite answer to question 1. Specified in `docs/spec/purchase/usecase.md`.

### Recording discipline

This classification depends on **requirements**, not on the structure of the operation. The
same operation changes classification when requirements change (e.g., if "stock restored by
response time" becomes a requirement, withdrawal moves to CommandService).

Therefore, record the **tolerance judgment as the rationale**, not the conclusion alone. Both
questions are recorded, because an operation that answers them differently is a different operation:

- Bad — "Withdrawal is a usecase."
- Good — "The withdrawal cascade tolerates eventual consistency; therefore it is a usecase."
- Bad — "Withdrawal locks the user row."
- Good — "A concurrent purchase would invalidate the withdrawal guard; therefore the user row is
  locked before the check."
- Good — "A product going unpublished cannot invalidate a purchase that already snapshotted its unit
  price; therefore that condition is left unguarded."

## Consequences

### Positive Consequences

- Placement decisions shift from subjective interpretation of inter-aggregate semantics to
  verifiable requirement checks.
- The direction of default (decomposition) vs. exception (CommandService) is explicit,
  suppressing misuse at the policy level.
- The definition is symmetric with QueryService, so both sides of CQRS are explained in the
  same grammar.
- Because the rationale is recorded as a requirement, the places that need re-evaluation are
  identifiable when requirements change.
- A cross-aggregate read now has a branch of its own, so "it is only a read" can no longer route an
  unheld condition onto the default path by omission.
- Every widening of the transaction boundary is named and bounded, and its relation to the
  aggregate-as-consistency-boundary principle is stated rather than left for a reader to reconstruct
  from the code.

### Negative Consequences

- The criterion requires an explicit requirements judgment ("can this cascade tolerate
  eventual consistency?") before implementation; the answer cannot be derived from code
  structure alone.
- Classification is requirement-relative: when requirements change, an operation may need to
  migrate between usecase + outbox and CommandService.
- The atomicity judgment and its recorded rationale must be maintained by review; there is no
  compiler enforcement for the distinction.
- The default path presumes the outbox machinery (relay process, at-least-once delivery), so
  its eventual-consistency effects must be acceptable to downstream consumers.
- Branch 2 costs throughput on the guarded row: the operation now blocks for the duration of any
  concurrent writer of that row ([ADR-0033](0033-ordered-pessimistic-row-locks.md)).
- Three branches are harder to hold than two, and two of them produce the same construct (a regular
  usecase), so the presence of a lock — not the class of the file — is what distinguishes branch 2
  from branch 1 at a glance.

## Alternatives Considered

### Classification by derivability

Classifying by "whether the change to secondary aggregates is derivable as a consequence of
the primary operation." This inserts a subjective judgment about inter-aggregate semantics,
so decisions vary by implementer. It also classifies purchase creation as a regular usecase
and withdrawal as a CommandService — the reverse of what the atomicity criterion yields —
evidence that the criterion is not grounded in requirements. Rejected.

### Classifying every multi-aggregate write as CommandService

Simpler to decide, but it creates an incentive to pull work that could be eventually
consistent into synchronous transactions, contradicting this project's outbox-first design
([ADR-0051](0051-transactional-outbox.md)). Rejected.

### Treating a cross-aggregate read as needing no mechanism

Keeping the two-way procedure and letting every guard fall to the default path, on the grounds that
a read changes nothing. Rejected. It is the reading that produced the gap: it answers a question
about atomicity that a guard never asks, and leaves the guard deciding from a condition nothing
holds — the interleaving [ADR-0033](0033-ordered-pessimistic-row-locks.md) documents. It also makes
the two directions of one invariant look like unrelated operations, since only the writing side gets
a mechanism.

### Promoting a guarded operation to CommandService

Routing anything that touches a second aggregate's row — lock included — through a CommandService,
so that "crosses the boundary" and "is a CommandService" coincide. Rejected. It restates the
already-rejected "spans multiple aggregates, therefore CommandService" with the lock as the trigger,
and it misplaces the work: taking a lock is a transaction-boundary responsibility, which the usecase
already owns, and the condition the lock protects belongs to the domain
([ADR-0033](0033-ordered-pessimistic-row-locks.md) § 5), not to a write-optimization construct.

### Raising the isolation level instead of adding a branch

Running every transaction at SERIALIZABLE, so that a stale guard surfaces as a serialization failure
and no per-operation judgment is needed. Rejected here for the reason
[ADR-0033](0033-ordered-pessimistic-row-locks.md) records: the isolation level is a property of every
transaction in the application, and its retry-rate cost is not a decision any single invariant should
make.

### Introducing Saga / process manager

The general solution for multi-aggregate writes, but at this project's scale, outbox
choreography is sufficient and an orchestration layer does not justify its complexity.
Rejected for now; not excluded as a future evolution.

## Notes

- **Relationship to [ADR-0029](0029-lightweight-cqrs.md)**: complementary, not conflicting.
  ADR-0029 defines the construct's structure (interface placement, implementation directory,
  DTO returns, DI registration); this ADR supplies the placement criterion ADR-0029 left
  open. One clarification: ADR-0029's consequence that CommandService "can freely optimize
  flexible updates, deletes" describes what the construct may do once placed, not when to
  place it. Under this ADR, write-shape flexibility or performance alone is NOT sufficient
  grounds for a CommandService; the atomicity requirement is the sole criterion, and this ADR
  governs placement going forward. ADR-0029 is not edited or superseded.
- **Relationship to [ADR-0033](0033-ordered-pessimistic-row-locks.md)**: this ADR decides *whether* a
  cross-aggregate condition needs to be held; ADR-0033 decides *how* it is held — lock order,
  acquisition point, lock mode, and who is allowed to author the guarded condition. Branch 2 is the
  entry point into it, and nothing in branch 2 overrides it.
- Which rows a given workflow locks, and which business rule each lock protects, is feature content
  rather than an architectural decision, so it is specified with the feature — in this repository the
  removable sample set (`docs/spec/purchase/`, `docs/spec/user/`), referenced by path rather than
  linked, because those files are deleted by `make setup-remove-sample-api` while this ADR stays.
- Related: [ADR-0030](0030-system-cqrs-dml-category.md) (`system_cqrs` category that carries
  the outbox DML, outside the CQRS split); [ADR-0051](0051-transactional-outbox.md) (the
  outbox pattern that the default decomposition path relies on);
  [ADR-0032](0032-transaction-retry-idempotent-callers.md) (retry semantics of the single
  transaction an atomic write runs in).
