---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [process, ai, review]
---

# ADR-0081: Use multi-model adversarial review with finder and verifier subagents

## Status

accepted

## Context

A code review performed by the same AI model that wrote the code has a structural blind
spot: the reviewer carries the same biases, assumptions, and errors as the implementer.
Reviews that only flag what the implementer already considered provide low marginal value.

A review workflow is needed that:

1. Uses a different model than the implementer so blind spots differ between the two.
2. Reduces false positives — a finding that sounds real but is refuted by context should
   not reach the report.
3. Covers runtime behavior (DI wiring, authentication middleware, real DB) that unit tests
   with mocked dependencies cannot exercise.
4. Distinguishes code correctness from comment quality, which requires different review
   criteria and different resolution actions.

## Decision

The `local-review` skill implements a multi-model adversarial review in the
**finder → verifier** shape:

**Finder stage (concurrent):** Four code lenses (`correctness`, `security`, `architecture`,
`runtime-gap`) each run as an independent `adversarial-reviewer` subagent. A dedicated
`comment-reviewer` subagent covers comment quality. All finders run concurrently. The
orchestrator enforces that reviewer agents run on a different model than the implementer —
the reviewer agents default to `sonnet`, and the orchestrator overrides them when the
session model would otherwise match.

**Verifier stage (concurrent):** Each surviving finding is independently verified by a
`review-verifier` subagent, which re-derives its conclusion from the source code rather than
trusting the finder. Findings are classified `CONFIRMED`, `PLAUSIBLE`, or `REFUTED`.
Refuted findings are dropped before the report.

**Runtime stage (orchestrator):** When a touched endpoint is detected, the orchestrator
performs a live `curl` and observability log check against the running application. This
exercises the Fx DI graph, HTTP middleware (authentication, OpenAPI validation), and the
real database — none of which mocked unit tests reach.

**Comment fix stage:** Confirmed comment quality findings are applied to the working tree by
the orchestrator (not a subagent) after user confirmation, followed by `make fix` and
`make lint`. The four code lenses are read-only; their surviving findings are posted to the
branch PR as inline review comments by default.

## Consequences

### Positive Consequences

- Bias reduction: reviewer and implementer run on different models, so the review surfaces
  findings the implementer's own model would miss.
- False-positive control: the verifier independently re-derives each finding from the code;
  plausible-but-wrong findings are classified `REFUTED` and dropped before the final report.
- Runtime coverage: the curl and observability stage catches DI misconfiguration, missing
  `security:` declarations, and SQL behavioral gaps that mocked tests do not exercise.
- Comment quality findings are applied automatically (with one confirmation), reducing
  friction for fixing narrating or tautological comments.

### Negative Consequences

- Multiple concurrent subagent spawns increase cost and latency compared to a single-pass
  review.
- The runtime stage requires a live application environment (`make serve` and `make db-init`);
  it is skipped when no endpoint is touched by the change.
- The model-divergence guarantee depends on the orchestrator correctly detecting and
  overriding when the session model matches the reviewer model default.

## Alternatives Considered

### Single-model review

Simple to invoke, but the same model that wrote the code reviews it. Structural blind spots
are shared between implementer and reviewer. Rejected because the primary value proposition
is bias reduction through model divergence.

### Human-only review

Provides genuine independence but is not available at every commit and does not scale with
AI-driven development pace. The `local-review` skill is positioned as a pre-PR complement
to human review, not a replacement.

### Automated linting only

`make lint` and `make fix` catch formatting and static analysis errors but do not reason
about correctness semantics, authorization, architecture violations, or runtime behavior.
Rejected as insufficient for a full code review pass.

## Notes

- Source: `.claude/skills/local-review/SKILL.md`, `.claude/agents/adversarial-reviewer.md`,
  `.claude/agents/review-verifier.md`.
- The `adversarial-reviewer` and `review-verifier` agent files declare `model: sonnet`
  in their frontmatter; the orchestrator overrides this via the `Agent` tool `model`
  parameter when the session model is also `sonnet`.
- Comment quality findings are the only findings auto-applied in Step 5.5; the four code
  lenses are reported-only (no auto-fix applied by the skill).
- Findings are posted to the branch PR as inline review comments by default; suppress with
  `--no-comment`, or pass `--no-apply` to suppress the comment auto-fix.
