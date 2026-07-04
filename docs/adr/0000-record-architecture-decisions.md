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

The project's technology rationale historically lived in a single file
(`docs/decisions.md`): a growing collation of ~8 decisions plus a dependency inventory.
Two problems surfaced:

- **In-place editing loses history.** Decisions were rewritten when reality changed (e.g.
  the observability section was rewritten when the `OBSERVABILITY_ENABLED` flag was
  removed). The record of *why we once chose the old design* was discarded.
- **Mixed content.** The file blended immutable decisions with a living dependency table
  that must track `go.mod` — forcing a drifting catalog into a "rationale" document, where
  it silently went stale.

This is a **template** repository: downstream users fork it and need to understand *why*
each choice was made, and to *supersede* individual choices with their own without editing
a shared monolith.

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
- Forks can override one decision by adding one ADR.
- Drifting catalogs (dependencies) live where drift is expected, not inside immutable records.

### Negative Consequences

- More files and cross-references to maintain (each ADR also needs a `docs/ja/adr/` mirror).
- Contributors must classify (decision vs rule vs inventory) before writing — a small upfront judgment cost.

## Alternatives Considered

### Keep the single `docs/decisions.md`

Rejected: in-place edits erase decision history, and the file already mixed immutable
decisions with a drifting inventory.

### One file, append-only decision log (no per-decision files)

Rejected: forks cannot cleanly supersede an individual decision, and a single file grows
unbounded and merge-conflicts.

## Notes

- Migration scope, the full ordered ADR list (~92), and per-ADR source references live in
  [the ADR log](README.md).
- The dependency inventory that lived in `docs/decisions.md` moves to `docs/reference/dependencies.md`.
