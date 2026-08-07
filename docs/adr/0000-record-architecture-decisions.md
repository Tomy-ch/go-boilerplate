---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [meta, documentation]
---

# ADR-0000: Record architecture decisions as ADRs

## Status

accepted

## Context

Collating a project's technology rationale into a single growing file has two problems:

- **In-place editing loses history.** Rewriting a decision when reality changes discards
  the record of *why the old design was once chosen*.
- **Mixed content.** Blending immutable decisions with a living inventory (e.g. a dependency
  table that must track `go.mod`) forces a drifting catalog into a "rationale" document,
  where it silently goes stale.

A decision record is written for a reader who was not in the room: they need to see *why*
each choice was made, and to be able to replace one choice without editing a document that
every other choice shares.

## Decision

Record each architecture decision as its own immutable file under `docs/adr/`, in
MADR-lite form (see [`template.md`](template.md)). Classify every candidate before it
lands:

- **decision** / **exclusion** → an ADR here (immutable; supersede by a new ADR).
- **rule** (a day-to-day enforced consequence) → stays in `docs/rules.md`, may link its ADR.
- **inventory** (a catalog that drifts with code, e.g. the dependency list) → a living
  reference (`docs/reference/dependencies.md`), never an ADR.

ADRs are numbered in dependency / foundational order (principles first), and superseding a
decision means adding a new `accepted` ADR and flipping the old one to `superseded`, not
editing its body.

## Consequences

### Positive Consequences

- Decision history is preserved; supersession is auditable.
- One decision can be overridden by adding one ADR, leaving the others untouched.
- Drifting catalogs (dependencies) live where drift is expected, not inside immutable records.

### Negative Consequences

- More files and cross-references to maintain (each ADR also needs a `docs/ja/adr/` mirror).
- Contributors must classify (decision vs rule vs inventory) before writing — a small upfront judgment cost.

## Alternatives Considered

### Keep the single `docs/decisions.md`

Rejected: in-place edits erase decision history, and the file already mixed immutable
decisions with a drifting inventory.

### One file, append-only decision log (no per-decision files)

Rejected: an individual decision cannot be superseded cleanly, and a single file grows
unbounded and merge-conflicts.

## Notes

- Migration scope, the full ordered ADR list (~92), and per-ADR source references live in
  [the ADR log](README.md).
- The dependency inventory that lived in `docs/decisions.md` moves to `docs/reference/dependencies.md`.
