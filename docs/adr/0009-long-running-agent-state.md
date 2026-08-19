---
status: accepted
date: 2026-08-18
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

The closed improvement loop of [ADR-0008](0008-agent-environment-alignment.md) needs observation
that outlives a single run. That observation is kept **outside the repository, in the issue
tracker** — not in `.agents/`. What it retains is the **compressed finding**: the friction
encountered, the control it implicates, and the measured effect of the change made in response.
What it never retains is a narrative of what an agent did. The test is reuse, not recency: a
compressed finding is consumed by a later run and by the human deciding whether to keep, simplify,
revise, delete, or revert a control, while a run log can only be re-read.

The repository keeps only the loop's **configuration** — schemas, thresholds, exclusion lists —
which is edited deliberately and reviewed like any other committed file. Locally produced
observation is buffered in the skill-owned, ignored `tmp/` artifact and sent to its issue at session
end, so ordinary development never dirties the working tree.

## Consequences

The repository retains durable knowledge, not a linear history of agent activity. A resume artifact
remains available only while its owning workflow needs it. Loop observation never becomes repository
state at all: the working tree stays clean while a session runs, and correcting or withdrawing a
mis-recorded finding costs an issue edit rather than a commit. Environment hooks may inspect a worktree, its vendor tree, and its
DB-slot marker, but must not acquire a slot or reinitialize databases: that operation is intentional,
potentially destructive work for the point at which DB work begins.

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

### Keep loop observation in `.agents/`

Rejected. Every session would dirty the working tree, and correcting or deleting a finding — which
the loop does routinely, since a finding is superseded as soon as its control changes — would cost a
commit. A store whose upkeep requires a git operation is a store that stops being kept up. The issue
tracker already provides editing, deletion, labelling, and cross-linking to the pull request that
acted on a finding.
