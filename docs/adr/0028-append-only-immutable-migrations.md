---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [persistence, migration]
---

# ADR-0028: Treat migrations as append-only and immutable

## Status

accepted

## Context

Database migrations record the full history of schema changes. The project uses golang-migrate,
which applies migration files in sequence and records **only the version number and a dirty flag**
in `schema_migrations` — it stores no checksum of the file it ran. Editing a migration that an
environment has already applied therefore produces no error at all: golang-migrate sees the version
as done and skips it. The environments that ran the original file keep the old schema, environments
created afterwards get the new one, and **nothing reports the difference**. The divergence surfaces
later, as a query that works on one machine and fails on another.

That silence is the reason the rule has to be a convention rather than something the tooling
guarantees. `migration-check.yaml` verifies the numbering (no duplicate versions, no gaps, every
`up` paired with a `down`) but does not compare file contents against what was applied, and no
other gate does either.

Beyond the drift, modifying applied migrations destroys the audit trail and makes it impossible to
reproduce the schema at any historical version — a property that matters for debugging incidents,
rolling back to a known state, and onboarding environments from scratch.

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
- Every environment (local, CI, staging, production) runs the same migration sequence,
  eliminating environment-specific schema drift — provided the rule is actually followed, since
  nothing detects a violation after the fact.

### Negative Consequences

- Correcting a mistake in an applied migration requires adding a new forward migration even
  for trivial fixes (e.g., a misspelled comment).
- The migration sequence grows monotonically; there is no compaction or squash step.
- The rule rests on review alone. A violation is invisible to golang-migrate and to CI, so it is
  caught only by someone noticing the diff — which is exactly why it is written down here.

## Alternatives Considered

### Allow editing applied migrations

Permits inline corrections, but an environment that already applied the original file never
receives the edit and is never told so — the schema silently diverges from a freshly created one.
Rejected: the convenience of in-place edits is not worth a difference between environments that
no tool and no reviewer is prompted to look for. When an edit is nonetheless made deliberately,
every environment holding the old schema has to be rebuilt (`make db-reinit DB=<name>`), and that
cost belongs to whoever chose the edit.

### Migration squashing

Collapse all applied migrations into a baseline schema file periodically. Reduces file
count but loses the ability to replay individual steps; new environments bootstrapped from
the baseline cannot roll forward to a specific historical version. Out of scope for the
current project lifecycle.

## Notes

- `migration-check.yaml` checks the **numbering** only — duplicate versions, gaps, and `up` / `down`
  pairing (see [ADR-0029](0029-sequential-migration-ids.md) for that discipline). Immutability
  itself has no CI gate.
- Source: [`docs/rules.md`](../rules.md) § "Database Migration";
  [`database/migrations/README.md`](../../database/migrations/README.md) § "Rules".
