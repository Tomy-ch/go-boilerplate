# Architecture Decision Records (ADR)

This directory holds the project's **architecture decisions**, one immutable record per
file, in [MADR-lite](https://adr.github.io/madr/) form.

> **Provisional staging.** During the `docs/decisions.md` migration these files live under
> `docs/design/adr/` for review. The agreed final home is top-level `docs/adr/` (see
> [`../adr-migration-plan.md`](../adr-migration-plan.md)). Paths below describe the final
> layout.

An ADR captures a single decision at a point in time: the context, the options weighed,
the choice, and its consequences. Superseding a decision does **not** mean editing its
ADR — it means adding a *new* ADR whose `Status` is `accepted` and marking the old one
`superseded`. The record of *why we once chose X* is preserved.

## What belongs here (and what does not)

| Kind | Example | Home |
| --- | --- | --- |
| **decision** — a choice among alternatives with lasting consequences | "Adopt onion architecture" | this dir (ADR) |
| **exclusion** — a deliberate "we intentionally do NOT do X" | "No in-application rate limiter" | this dir (ADR) |
| **rule** — a day-to-day enforced constraint / consequence of a decision | "Controller must not import infrastructure" | `docs/rules.md` (may link an ADR) |
| **inventory** — a catalog that drifts with the code | the direct-dependency table | `docs/reference/dependencies.md` (living doc) |

The dependency inventory is **not** an ADR: it tracks `go.mod` and changes continuously,
which is the opposite of an immutable record. The *policy* for selecting dependencies is
a decision (an ADR); the *list* of them is a living reference.

## Conventions

- **Filename**: `NNNN-kebab-title.md`, zero-padded 4 digits, monotonically increasing. Numbers are never reused, even after supersession.
- **Ordering**: numbers follow dependency / foundational order (principles → contract → layers → subsystems → cross-cutting → exclusions), not discovery order.
- **Status lifecycle**: `proposed` → `accepted` → (`superseded` | `deprecated`).
- **Immutable**: once `accepted`, edit only the `Status` line and add a `Superseded-by` link. Everything else stays as written.
- **Template**: copy [`template.md`](template.md).
- **Meta**: [`0000-record-architecture-decisions.md`](0000-record-architecture-decisions.md) records the decision to use ADRs and this classification.
- **Translation**: each ADR mirrors to `docs/ja/adr/` (via the `canonicalize-doc` flow).
- **Exclusion ADRs** (deliberate "we do NOT do X") carry a `setup-review` tag so the repository-setup flow can enumerate them. At initial setup a fork may **edit these directly** to establish its own baseline; the supersede-by-new-ADR model applies only to changes made later. See `docs/get-started/setup-repository.md` Phase 10.

## Log

| # | Decision | Status |
| --- | --- | --- |
| [0000](0000-record-architecture-decisions.md) | Record architecture decisions as ADRs | accepted |
| [0001](0001-avoid-lock-in.md) | Adopt lock-in avoidance as a design principle | accepted |
| [0002](0002-onion-architecture.md) | Adopt pragmatic onion architecture | accepted |

<!-- The remaining ~90 decisions (0003 onward) are enumerated, ordered, and sourced in
../adr-migration-plan.md and are materialized in Phases 1-4 of that plan. -->

Frontmatter fields: `status`, `date`, `deciders`, `supersedes` / `superseded-by`, `tags`.
Consequences follow the MADR standard (`Positive` / `Negative`; optional `Neutral`).
