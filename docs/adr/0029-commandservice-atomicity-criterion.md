---
status: accepted
date: 2026-07-17
deciders: [maintainers]
tags: [persistence, cqrs, architecture]
---

# ADR-0029: Reserve CommandService for multi-aggregate writes that require single-transaction atomicity

## Status

accepted

## Context

[ADR-0027](0027-lightweight-cqrs.md) defines the *structure* of the CommandService
construct: its interface lives in the usecase layer (`internal/usecase/<aggregate>/command/`),
its implementation lives under `internal/infrastructure/rdb/command_service/<aggregate>/`, it
returns DTOs, and it is registered in `persistenceModule` and injected via DI. What ADR-0027
deliberately leaves open is the *placement criterion*: when does a write operation actually
warrant a CommandService instead of a regular usecase composed of Repository calls?

Without such a criterion, the placement of multi-aggregate write operations falls back to
implementer judgment about inter-aggregate semantics, and decisions vary by implementer. The
naive heuristic — "it touches more than one aggregate, so it needs a CommandService" — pulls
work that could be eventually consistent into synchronous transactions, contradicting this
project's outbox-first design ([ADR-0044](0044-transactional-outbox.md)). A related ambiguity
exists in ADR-0027 itself: its consequences note that CommandService "can freely optimize
flexible updates, deletes", which could be misread as write-shape flexibility or performance
being sufficient grounds for introducing one.

The goal is to decide where multi-aggregate write operations belong based on a verifiable
criterion rather than implementer judgment.

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

**Does the multi-aggregate write require single-transaction atomicity?**

Immediacy — all effects being visible at API response time — is the typical reason this
requirement arises.

### Decision procedure

When encountering a write that spans multiple aggregates:

1. First, ask the requirements whether it can be decomposed into eventual consistency (a
   cascade via outbox events, [ADR-0044](0044-transactional-outbox.md)).
2. If it can, implement it as a regular usecase + outbox (**default**).
3. Only when atomicity remains as a requirement, use a CommandService (**exception; must be
   justified**).

The default is decomposition; CommandService is a justified exception. "It spans multiple
aggregates, therefore CommandService" is not an acceptable justification.

### Illustrative scenarios

The repository currently has only `user` and `prefecture` aggregates and no purchase feature,
and the CommandService implementation itself is a reserved placeholder (see
[ADR-0027](0027-lightweight-cqrs.md)). The following are therefore worked hypotheticals that
demonstrate how the criterion is applied, not descriptions of existing endpoints.

- **Purchase creation (positive example: CommandService).** Stock validation and decrement
  must be atomic with purchase confirmation. A state where "a purchase succeeded without
  stock" must never be observable, even momentarily (no overselling). The writes to
  `purchases` / `purchase_details` / `products` therefore require single-transaction
  atomicity, and the outbox insert joins that same transaction as usual per
  [ADR-0044](0044-transactional-outbox.md). There is no room for decomposition. (The outbox
  insert itself is not what justifies the CommandService — a regular usecase also writes the
  outbox in its own transaction; the justification is the stock/purchase atomicity.)
- **User withdrawal (negative example: avoided by decomposition).** The core of withdrawal is
  a single-aggregate write to `users.deleted_at`. The cascade — cancelling pending purchases
  and restoring stock — requires no immediacy and is handled as eventual consistency via
  outbox events. The invariant check "cannot withdraw with purchases in progress" does span
  aggregates, but it is a read and requires no write atomicity. It is therefore implemented
  as a regular usecase + outbox.

### Recording discipline

This classification depends on **requirements**, not on the structure of the operation. The
same operation changes classification when requirements change (e.g., if "stock restored by
response time" becomes a requirement, withdrawal moves to CommandService).

Therefore, record the **tolerance judgment as the rationale**, not the conclusion alone:

- Bad — "Withdrawal is a usecase."
- Good — "The withdrawal cascade tolerates eventual consistency; therefore it is a usecase."

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
([ADR-0044](0044-transactional-outbox.md)). Rejected.

### Introducing Saga / process manager

The general solution for multi-aggregate writes, but at this project's scale, outbox
choreography is sufficient and an orchestration layer does not justify its complexity.
Rejected for now; not excluded as a future evolution.

## Notes

- **Relationship to [ADR-0027](0027-lightweight-cqrs.md)**: complementary, not conflicting.
  ADR-0027 defines the construct's structure (interface placement, implementation directory,
  DTO returns, DI registration); this ADR supplies the placement criterion ADR-0027 left
  open. One clarification: ADR-0027's consequence that CommandService "can freely optimize
  flexible updates, deletes" describes what the construct may do once placed, not when to
  place it. Under this ADR, write-shape flexibility or performance alone is NOT sufficient
  grounds for a CommandService; the atomicity requirement is the sole criterion, and this ADR
  governs placement going forward. ADR-0027 is not edited or superseded.
- Related: [ADR-0028](0028-system-cqrs-dml-category.md) (`system_cqrs` category that carries
  the outbox DML, outside the CQRS split); [ADR-0044](0044-transactional-outbox.md) (the
  outbox pattern that the default decomposition path relies on);
  [ADR-0030](0030-transaction-retry-idempotent-callers.md) (retry semantics of the single
  transaction an atomic write runs in).
