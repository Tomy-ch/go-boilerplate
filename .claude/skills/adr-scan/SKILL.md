---
name: adr-scan
description: PROVISIONAL / one-off. Full-repository scan that discovers ADR-worthy architectural decisions across the whole repo (not just docs/decisions.md) so decisions.md can be migrated to a formal docs/adr/ set. Fans out read-only discovery over four surfaces — canonical docs (decisions/architecture/rules), subsystem design docs (docs/design/**), deliberate-exclusion docs (docs/project/**, out-of-scope), and latent decisions (per-package READMEs + code "why" comments) — classifying each candidate as decision (ADR-worthy) / rule (stays in rules.md) / inventory (stays in a living reference) / exclusion (ADR-worthy negative decision). Read-only: produces a candidate ADR inventory only, writes no docs/adr files. Delete after the migration format is locked.
---

# adr-scan (provisional)

**Status: temporary.** Built to drive the one-off `decisions.md` → `docs/adr/` migration
survey. Remove once the ADR skeleton/numbering is agreed. Read-only — never writes
`docs/adr/**` or edits source; it only emits a candidate inventory.

## Purpose

`docs/decisions.md` holds 8 explicit decisions, but architectural decisions are also
scattered across design docs, rules, deliberate exclusions, and code comments. Before
migrating to a formal per-file ADR set, discover the full ADR-worthy surface so the
numbering and split are done once.

## Classification taxonomy (the core judgment)

Each candidate is exactly one of:

- **decision** — a choice among alternatives with lasting consequences (X over Y). ADR-worthy.
- **exclusion** — a deliberate "we intentionally do NOT do X" with rationale. ADR-worthy (negative decision).
- **rule** — a consequence/constraint enforced day-to-day (layer deps, DTO boundary). Stays in `docs/rules.md`; may *reference* an ADR but is not itself one.
- **inventory** — a catalog that drifts with code (dependency table). Stays in a living reference doc, never an ADR.

A candidate is ADR-worthy only if: it has (or implies) considered alternatives, it is
cross-cutting or hard to reverse, and it is not merely restating a rule/inventory.

## Scan surfaces (fan out one worker each)

1. **Canonical** — `docs/decisions.md`, `docs/architecture.md`, `docs/rules.md`. Extract explicit decisions + rule-encoded decisions; separate decision vs rule.
2. **Subsystem design** — `docs/design/*.md` (idempotency / job / observability / outbox / rest / worker). Each design doc's "why / alternatives / trade-off" content.
3. **Deliberate exclusions** — `docs/project/*.md` (esp. `out-of-scope.md`), `policy.md`, `scope.md`. Negative decisions with rationale.
4. **Latent** — per-package `README.md` (Design Intent / Notes) + code `// why` rationale comments. Decisions made in code but never elevated to a doc.

## Output (per candidate)

```text
- title:        short decision title (imperative, ADR-style)
  type:         decision | exclusion | rule | inventory
  adr_worthy:   yes | no
  source:       file:line (evidence)
  alternatives: present | implied | none
  proposed_adr: which ADR bucket it belongs to (or existing 0001..N)
  note:         one line — why worthy / why not
```

Aggregate into: (A) ADR bucket list with proposed numbering, (B) rules-stay list,
(C) inventory-stay list, (D) newly-discovered decisions not in decisions.md.
