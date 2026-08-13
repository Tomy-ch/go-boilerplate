---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [dependencies, policy]
---

# ADR-0075: Single-responsibility library selection policy

## Status

accepted

## Context

To keep the dependency surface auditable and replaceable (a direct consequence of the
lock-in-avoidance principle, [ADR-0001](0001-avoid-lock-in.md)), the project needs a stated
rule for *when* a third-party library may be adopted — otherwise dependencies accrete by
convenience and the surface becomes hard to reason about or swap.

## Decision

Adopt a library only when it satisfies **one responsibility = one concern, ideally bound to
a single upstream ecosystem**. Each direct dependency must map to a single, nameable,
replaceable responsibility.

Libraries that stand between **two independently-versioned upstreams** (a framework/library
*and* OpenTelemetry) are **bridge / instrumentation** libraries; they deviate from the
single-responsibility criterion and are treated as explicit, individually-justified
exceptions (see [ADR-0076](0076-bridge-instrumentation-exceptions.md)).

## Consequences

### Positive Consequences

- The dependency surface stays auditable — each library has one replaceable job.
- Replacing any single concern has a bounded blast radius.
- A clear yes/no test for proposed dependencies, reducing incidental accretion.

### Negative Consequences

- Some convenient multi-purpose libraries are declined in favour of narrower ones.
- Genuine glue code (bridges) needs an explicit exception path rather than a blanket rule.

## Alternatives Considered

### No stated policy (adopt per convenience)

Rejected: without a criterion, dependencies accumulate incidentally and the surface loses
auditability and replaceability.

### Ban all multi-upstream libraries outright

Rejected: instrumentation/bridge glue is genuinely useful and hand-rolling it is worse;
these are admitted as bounded, documented exceptions instead.

## Notes

- Parent principle: [ADR-0001](0001-avoid-lock-in.md). Exceptions: [ADR-0076](0076-bridge-instrumentation-exceptions.md).
- The *list* of dependencies is an inventory, not a decision — it lives in `docs/reference/dependencies.md`; this ADR records the *policy* only.
- Migrated from the former `docs/decisions.md`.
