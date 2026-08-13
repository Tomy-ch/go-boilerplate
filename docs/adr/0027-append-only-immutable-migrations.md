---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [persistence, migration]
---

# ADR-0027: Treat migrations as append-only and immutable

## Status

accepted

## Context

Database migrations record the full history of schema changes. The project uses
golang-migrate, which applies migration files in sequence and can detect tampering via
checksums. If a migration file that has already been applied to an environment is modified,
golang-migrate reports a hash mismatch and refuses to proceed, breaking all environments
that previously ran the original file.

Beyond tool enforcement, modifying applied migrations destroys the audit trail and makes it
impossible to reproduce the schema at any historical version — a property that matters for
debugging incidents, rolling back to a known state, and onboarding environments from
scratch.

## Decision

All migration files under `database/migrations/` are **append-only and immutable**. Once a
file has been committed, it must never be modified. All schema changes — including
corrections, column renames, and constraint alterations — must be expressed as new migration
files. See [`docs/rules.md`](../rules.md) § "Database Migration" for the authoritative
rule and rationale.

## Consequences

### Positive Consequences

- The schema at any point in history is reproducible by replaying the migration sequence
  from the beginning.
- golang-migrate hash verification provides an automatic integrity check against accidental
  edits.
- Every environment (local, CI, staging, production) runs the same migration sequence,
  eliminating environment-specific schema drift.

### Negative Consequences

- Correcting a mistake in an applied migration requires adding a new forward migration even
  for trivial fixes (e.g., a misspelled comment).
- The migration sequence grows monotonically; there is no compaction or squash step.

## Alternatives Considered

### Allow editing applied migrations

Permits inline corrections but invalidates golang-migrate's hash verification. Environments
that already applied the original file break silently or fail on the next run. Rejected —
the integrity guarantee is more valuable than the convenience of in-place edits.

### Migration squashing

Collapse all applied migrations into a baseline schema file periodically. Reduces file
count but loses the ability to replay individual steps; new environments bootstrapped from
the baseline cannot roll forward to a specific historical version. Out of scope for the
current project lifecycle.

## Notes

- Enforced in CI by `migration-check.yaml` (see [ADR-0028](0028-sequential-migration-ids.md)
  for the numbering discipline).
- Source: [`docs/rules.md`](../rules.md) § "Database Migration";
  [`database/migrations/README.md`](../../database/migrations/README.md) § "Rules".
