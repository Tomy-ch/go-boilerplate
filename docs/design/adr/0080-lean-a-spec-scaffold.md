---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [process, ai, scaffold]
---

# ADR-0080: Scaffold only domain and usecase from spec files; derive controller and infra from generated code

## Status

accepted

## Context

Scaffolding an onion-architecture endpoint requires human input for the parts of the design
that cannot be mechanically derived: domain invariants, behavior method semantics, value
object validations, and usecase workflow steps. However, the controller and infrastructure
layers follow deterministic templates once their generated inputs exist:

- A controller handler is a pure mapping from an OpenAPI `ServerInterface` method to a
  usecase method call. Its shape is fully determined by the generated interface and a
  naming convention heuristic.
- A Repository implementation is a pure mapping from a domain Repository interface method
  to a sqlc-generated function. Its shape is fully determined by the domain interface and
  the sqlc output.

Writing separate spec files for controller and infra would duplicate the OpenAPI YAML and
sqlc gen content, creating a second source of truth that can drift silently. It would also
impose authoring cost without adding derivation value, since the derivation is deterministic
given the generated code and naming convention.

## Decision

The scaffold tooling follows a **lean-A constitution**: only `domain.md` and `usecase.md`
spec files are required under `docs/spec/<feature>/`. Controller and infrastructure layers
are derived from generated code and naming conventions — no `controller.md` or `infra.md`
spec files exist.

- `scaffold-domain` consumes `domain.md` (sections: Overview, Entity, Cross-field
  Invariants, Behavior Methods, Value Objects, Repository Methods) to generate the entity,
  value objects, constants, errors, and Repository interface.
- `scaffold-usecase` consumes `usecase.md` (sections: Overview, Interface, DTOs,
  Dependencies, Workflow) to generate the Application Service, DTOs, and boundary wiring.
- `scaffold-controller` derives the handler from the OpenAPI-generated `ServerInterface`
  and the usecase Interface via name-match heuristic. It halts with a hand-off message if
  an `operationId` cannot be mapped to a usecase method — no auto-resolution.
- `scaffold-infra-db` derives the Repository from the domain Repository interface and the
  sqlc-generated functions via name-match heuristic. For unmapped methods it emits TODO
  stubs rather than halting, because partial Repository compilation remains valid.
- `scaffold-endpoint` orchestrates all four in dependency order after `verify-spec`
  validates the two spec files.

Correctness of controller and infra against their derivation rules is enforced by
`arch-check`, which serves as the safety net when a naming-convention template is violated.

## Consequences

### Positive Consequences

- Reduced authoring burden: engineers write two spec files that cover the non-derivable
  design choices, not four.
- Single source of truth for controller and infra shape: the generated code (OpenAPI gen,
  sqlc gen) is the definition; spec duplication and drift are structurally prevented.
- Spec files cover only the decisions requiring design judgment (invariants, behavior,
  workflow), making them useful review artifacts rather than mechanical transcriptions.
- `arch-check` provides automated enforcement of controller/infra templates without
  requiring a spec file.

### Negative Consequences

- Controller scaffolding halts when an `operationId`-to-usecase name mapping cannot be
  derived. Naming mismatches require manual resolution before scaffold can proceed.
- Infra correctness depends on `arch-check` running after scaffold rather than on an
  upfront spec. A violated naming convention produces a TODO stub rather than an early
  error.
- The lean-A constitution requires OpenAPI YAML, SQL migrations, and sqlc gen to exist
  before scaffold runs; these are human-written preconditions that must be satisfied first.

## Alternatives Considered

### Four-layer spec (domain + usecase + controller + infra)

Explicit but redundant. Controller and infra spec content would duplicate the OpenAPI YAML
and sqlc gen output, creating a second source of truth that drifts silently as the generated
files evolve. Rejected because the derivation is deterministic enough to not warrant a
separate spec.

### No spec files (pure derivation)

Would require the scaffold to infer domain invariants, validation rules, behavior method
semantics, and usecase workflow orchestration from the SQL schema and OpenAPI spec alone.
Domain invariants and usecase Workflow are design decisions that cannot be reliably derived
from data schema definitions. Rejected because the spec captures design judgment that is
genuinely non-derivable from the generated artifacts.

## Notes

- Source: `.claude/scaffold-spec/lifecycle.md` (§ "なぜ 2 spec か"),
  `.claude/skills/scaffold-endpoint/SKILL.md`,
  `.claude/skills/scaffold-controller/SKILL.md`,
  `.claude/skills/scaffold-infra-db/SKILL.md`,
  `.claude/scaffold-spec/domain-spec.md`,
  `.claude/scaffold-spec/usecase-spec.md`.
- Spec files are committed to the repository under `docs/spec/<feature>/` as permanent
  design artifacts; they remain after scaffold completes and are reviewed alongside the PR.
- Naming convention enforcement for controller and infra is governed by `arch-check`; see
  [`docs/rules.md`](../../rules.md) for the layer dependency and purity rules that underpin it.
- Controller derivation halts on unmapped `operationId`; infra derivation continues with
  TODO stubs. This asymmetry reflects that a partial infra impl compiles, while a partial
  handler impl would violate the generated `ServerInterface`.
