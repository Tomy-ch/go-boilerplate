---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [persistence, migration, ci]
---

# ADR-0023: Use sequential 6-digit migration IDs with CI-enforced gap and pair checks

## Status

accepted

## Context

golang-migrate applies migrations in filename order. Two common ID schemes are timestamps
(e.g., `20240101120000_`) and sequential integers. Timestamp IDs avoid numbering conflicts
between parallel branches but are verbose, harder to read at a glance, and still require
CI verification of pairs. Sequential integer IDs are compact and unambiguous in ordering,
but parallel branches can accidentally claim the same number — a collision that must be
caught before merge.

Without tooling enforcement, any ID scheme can accumulate gaps (a deleted or skipped file),
missing down files (making rollback impossible), or duplicate IDs (undefined apply order).
Silent failures in these cases are costly in production.

## Decision

Migration files use a zero-padded **6-digit sequential integer** prefix starting at
`000001`, in the format `{6-digit sequence}_{description}.{up|down}.sql`. Every migration
must be created as a **up/down pair** using `make new-migrate-<name>`, which auto-generates
the next number.

The `migration-check.yaml` CI workflow runs on every pull request that touches
`database/migrations/**` and enforces:

- No duplicate sequence numbers (separate checks for `.up.sql` and `.down.sql`)
- No gaps in the sequence for either set
- Every up file has a matching down file (pair completeness check via diff)

A PR that violates any of these checks fails CI and cannot merge. See
[ADR-0022](0022-append-only-immutable-migrations.md) for the immutability rule that this
numbering discipline supports.

## Consequences

### Positive Consequences

- Application order is unambiguous and human-readable (numeric sort = apply order).
- A numbering collision between parallel branches fails CI immediately, forcing coordination
  before merge rather than after deployment.
- CI guarantees that every environment can apply the full sequence and roll back any step.
- The 6-digit zero-padding keeps filenames lexicographically sortable up to 999,999
  migrations — sufficient for any realistic project lifetime.

### Negative Consequences

- Parallel feature branches that both add migrations must coordinate sequence number
  assignment; the second branch to open a PR must renumber if there is a conflict.
- The CI check runs only on PR files touching `database/migrations/**`; local development
  must use `make new-migrate-<name>` to avoid manually picking numbers.

## Alternatives Considered

### Timestamp-based IDs

Avoids numbering collisions between branches because timestamps are naturally unique per
developer per second. However, timestamps are verbose (14 characters vs 6), sort less
cleanly in directory listings, and still require gap and pair checks. The project prefers
the compactness of sequential integers.

### 4-digit IDs

Simpler but collide at 9,999 migrations. The 6-digit convention is already established and
provides ample headroom.

### No CI enforcement (honor system)

Relies on developer discipline. Migration issues discovered after deployment are expensive
to fix under the immutability constraint. CI enforcement is non-negotiable.

## Notes

- Source: [`database/migrations/README.md`](../../database/migrations/README.md) § "File
  Naming Convention" and § "CI Check".
- CI workflow: [`.github/workflows/migration-check.yaml`](../../.github/workflows/migration-check.yaml).
- Related: [ADR-0022](0022-append-only-immutable-migrations.md) (immutability).
