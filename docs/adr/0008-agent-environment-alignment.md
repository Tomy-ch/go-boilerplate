---
status: accepted
date: 2026-08-10
deciders: [maintainers]
tags: [process, ai, architecture]
---

# ADR-0008: Align the agent environment around declared, checkable properties

## Status

accepted

## Context

This repository already constrains agent work through canonical documentation, protected paths,
layer rules, generated-artifact gates, skills, reviews, and runtime checks. Those mechanisms were
adopted individually, but no decision states the shared intent or how a later maintainer should
judge a proposed control.

The external phrase "harness engineering" describes similar practices, but its terminology and
definition are still unsettled. Binding this long-lived template to that vocabulary would create a
new form of lock-in without improving verification.

## Decision

We declare alignment around the repository's own checkable properties, rather than compliance with
an external label. A control must provide a clear instruction, mechanical enforcement where the
property is decidable, or an independently reviewable signal where it is not.

Existing mechanisms remain the source of their individual rules. In particular, deterministic
properties gate through tooling; reading-comprehension judgments use the finder-to-verifier review
shape of ADR-0090. The `ci-first` load band deliberately delegates heavy local gates to CI when a
saturated host would make their failures untrustworthy. Signals that reliably reappear through an
existing mechanism need not be escalated as durable human work items.

## Consequences

### Positive Consequences

- New controls can be judged against explicit properties without adopting a volatile external term.
- Documentation, skills, hooks, and CI remain complementary rather than competing stores of policy.
- The boundary between mechanical gate and human judgment stays explicit.

### Negative Consequences

- Alignment does not claim conformance to any external checklist.
- Later work must keep the interpretation and maintenance artifacts synchronized with this decision.

## Alternatives Considered

### Declare compliance with "harness engineering"

Rejected because the external term is unsettled and the declaration would outlive its current
meaning.

### Make no umbrella declaration

Rejected because the existing mechanisms would remain difficult to evaluate as one coherent system.

## Notes

- Related mechanisms: ADR-0006, ADR-0007, ADR-0086, and ADR-0090.
- The repository's interpretation is documented separately under `docs/design/`.
