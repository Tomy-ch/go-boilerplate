---
status: accepted
date: 2026-08-18
deciders: [maintainers]
tags: [process, ai, architecture]
---

# ADR-0008: Align the agent environment around declared, checkable properties, and improve it in a closed loop

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

A second gap appears once AI-assisted development becomes the standard path
([ADR-0007](0007-agents-md-operational-contract.md)): controls are added far more readily than they
are removed. A skill nobody invokes, a rule that fires only on cases that no longer occur, and a
document that costs a read on every session are indistinguishable from useful ones as long as
nothing measures them. An agent environment that only grows becomes the thing it was built to
prevent — a structure too large to hold.

## Decision

We declare alignment around the repository's own checkable properties, rather than compliance with
an external label. A control must provide a clear instruction, mechanical enforcement where the
property is decidable, or an independently reviewable signal where it is not.

Existing mechanisms remain the source of their individual rules. In particular, deterministic
properties gate through tooling; reading-comprehension judgments use the finder-to-verifier review
shape of ADR-0092. The `ci-first` load band deliberately delegates heavy local gates to CI when a
saturated host would make their failures untrustworthy. Signals that reliably reappear through an
existing mechanism need not be escalated as durable human work items.

**The agent environment is not append-only.** Every control it holds — skill, rule, document, CI
step, tooling — carries a lifecycle rather than only an introduction:

1. AI development sessions are observed.
2. The friction they hit is collected and semantically compressed into feedback.
3. That feedback is attributed to the skill, rule, documentation, CI check, or tool it implicates.
4. Improvement candidates are generated from it.
5. A human takes the design decisions the change requires.
6. The improvement lands.
7. After an interval, its effect is re-evaluated against the friction it was meant to remove.
8. The control is then kept, simplified, revised, deleted, or reverted.

Skills are inventoried on the same terms — usage frequency, the occasions on which they would have
applied, the friction they caused, and the measured effect of the last change to them.

**The loop does not grant an agent unbounded self-modification.** The design decisions inside it
stay with a human (ADR-0007, point 4), and the AI-tool configuration directories remain outside an
agent's default modification scope: they are reachable only through an explicitly invoked skill,
under the skill-execution exception `AGENTS.md` defines and bounded by that skill's own procedure.
The purpose is to return what AI-assisted development reveals back into the development foundation
itself, not to let the foundation rewrite itself.

## Consequences

### Positive Consequences

- New controls can be judged against explicit properties without adopting a volatile external term.
- Documentation, skills, hooks, and CI remain complementary rather than competing stores of policy.
- The boundary between mechanical gate and human judgment stays explicit.
- Removing a control becomes an evidenced decision rather than a nerve-holding one: a control that
  stopped paying for itself has a record saying so.

### Negative Consequences

- Alignment does not claim conformance to any external checklist.
- Later work must keep the interpretation and maintenance artifacts synchronized with this decision.
- The loop needs durable observation data of its own. It lives in the issue tracker rather than in
  the repository, so it never becomes repository state;
  [ADR-0009](0009-long-running-agent-state.md) draws that line and gives the reason a
  git-maintained store is the wrong one for data that is corrected and retired routinely.
- Re-evaluation is work that has to actually happen. A loop whose step 7 is skipped degrades into
  the append-only accumulation it was adopted to prevent.

## Alternatives Considered

### Declare compliance with "harness engineering"

Rejected because the external term is unsettled and the declaration would outlive its current
meaning.

### Make no umbrella declaration

Rejected because the existing mechanisms would remain difficult to evaluate as one coherent system.

### Add controls freely and prune them when someone notices

Rejected. Nobody notices: a skill that is never invoked produces no failure, so the signal that it
should be removed is exactly the absence of a signal. Without a declared re-evaluation step, the
environment only grows.

### Let the loop apply its own improvements end to end

Rejected. The decisions the loop surfaces are architecture, domain, and policy decisions, which
ADR-0007 keeps behind a human gate. An automated loop closing over its own rules would make the
contract self-modifying and remove the reason to trust it.

## Notes

- Related mechanisms: ADR-0006, ADR-0007, ADR-0009, ADR-0088, and ADR-0092.
- The repository's interpretation is documented separately under
  [`docs/design/agent-environment.md`](../design/agent-environment.md).
