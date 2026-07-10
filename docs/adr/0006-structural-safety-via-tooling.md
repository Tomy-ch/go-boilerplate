---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [foundational, ci, structural-safety]
---

# ADR-0006: Enforce structural safety with tooling and CI (depguard)

## Status

accepted

## Context

Layer dependency rules are documented in [`docs/rules.md`](../rules.md) and summarized
in `AGENTS.md`. Documentation alone is insufficient: rules that rely on code-review
discipline are inconsistently applied, especially as the codebase grows and contributors
change. A cross-layer import that slips through review — for example, a usecase package
importing an infrastructure package — silently erodes the architectural boundaries that
[ADR-0002](0002-onion-architecture.md) and [ADR-0003](0003-interface-based-decoupling.md)
establish.

This project makes a deliberate choice to encode structural rules as machine-checkable
constraints so they cannot be bypassed by accident. The same principle applies to other
structural concerns: generated code must never be manually edited (verified by CI
regeneration checks), and API contracts must precede implementation (the OpenAPI-first flow).

The design goals are maintainability, predictability, and structural safety — not raw
performance or minimal tooling. Investing in tooling infrastructure is consistent with the
long-term operability objective. Critically, the same CI gate applies to human-authored and
AI-generated code alike: an agent that violates a boundary receives the same build failure
as a human contributor.

## Decision

Enforce layer dependency boundaries using `golangci-lint` with the `depguard` linter.
Forbidden cross-layer imports cause CI to fail. The four core depguard rule sets, configured
in `.golangci.yaml`, are:

- `maintain_a_sound_domain` — domain may not import usecase, controller, or infrastructure
  packages; nor I/O-side `pkg/` utilities (filesystem, process execution, env-write).
- `maintain_a_sound_usecase` — usecase may not import controller or infrastructure packages;
  nor I/O-side `pkg/` utilities.
- `maintain_a_sound_controller` — controller may not import infrastructure packages.
- `maintain_a_sound_infrastructure` — infrastructure may not import controller packages.

The full consequence table (what is and is not allowed per layer) is in
[`docs/rules.md`](../rules.md); this ADR records only the decision to enforce those rules
via tooling rather than documentation alone.

## Consequences

### Positive Consequences

- Layer violations are caught at CI time, not during code review — detection is objective and
  continuous.
- AI-generated code that violates boundaries fails the same CI gate as human-authored code.
- The linter configuration in `.golangci.yaml` is the machine-readable authority on permitted
  imports per layer; it is version-controlled alongside the code it governs.

### Negative Consequences

- Adding a legitimate new `pkg/` utility that should be accessible inside a restricted layer
  requires a conscious update to `.golangci.yaml` — intentional friction, but friction
  nonetheless.
- Depguard operates on import paths; it cannot detect architectural violations that stay
  within a single package (e.g., business logic written inside an infrastructure file that
  imports no forbidden packages).

## Alternatives Considered

### Documentation and code-review only

Rules written in `docs/rules.md` and enforced through pull-request review alone. Rejected:
inconsistent across reviewers and over time; AI agents have no runtime feedback loop for
violation detection.

### Go workspace modules per layer

Each layer as a separate Go module, making cross-layer imports fail at `go build`. Provides
stronger isolation but adds significant module-management overhead (replace directives,
multi-module CI). The depguard approach achieves the same practical effect with much lower
complexity.

### Custom go/analysis pass

Write a custom static-analysis pass. More flexible but requires authoring and maintaining
a bespoke linter. Depguard covers the use case without custom code.

## Notes

- Full layer rules and rationale: [`docs/rules.md`](../rules.md) §§ "Layer Dependency
  Rules", "Usecase Dependency Rules", "Domain Layer Constraints", "Infrastructure
  Implementation Rules".
- Linter configuration: `.golangci.yaml` (depguard `rules` block —
  `maintain_a_sound_domain`, `maintain_a_sound_usecase`, `maintain_a_sound_controller`,
  `maintain_a_sound_infrastructure`).
- Source: `docs/architecture.md` § "Structural Safety".
- Source: `docs/rules.md` § "Layer Dependency Rules" (enforcement note).
- Related: [ADR-0002](0002-onion-architecture.md) (layer shape),
  [ADR-0003](0003-interface-based-decoupling.md) (interface seams).
