---
status: accepted
date: 2026-08-10
deciders: [maintainers]
tags: [process, ai]
---

# ADR-0009: Keep durable agent state in its owning canonical form

## Decision

Do not accumulate progress logs as repository state. A finding that changes a describing document is
returned to its owning README or `docs/` reference; a conflict with a governing document is raised to
the architect or tech lead. Per-run state stays in the skill-specific, ignored `tmp/` artifact that
owns its restart semantics. It is not moved to `.agents/`, and the distinct formats of `impl-issue`,
`full-verify`, and `full-apply` are not unified.

## Consequences

The repository retains durable knowledge, not a linear history of agent activity. A resume artifact
remains available only while its owning workflow needs it. Environment hooks may inspect a worktree,
its vendor tree, and its DB-slot marker, but must not acquire a slot or reinitialize databases: that
operation is intentional, potentially destructive work for the point at which DB work begins.

## Alternatives Considered

### Persist every run's progress

Rejected because it creates an ever-growing second documentation system whose audience and authority
are unclear.

### Put all resume state in `.agents/`

Rejected because `.agents/` is committed shared state, while resume artifacts are per-run operator
state.

### Standardize the three existing resume formats

Rejected because approved-plan reconciliation, idempotent review enumeration, and apply/defer
decisions have different sources of truth.
