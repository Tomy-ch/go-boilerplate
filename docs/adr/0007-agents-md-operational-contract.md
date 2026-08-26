---
status: accepted
date: 2026-08-18
deciders: [maintainers]
tags: [foundational, process, ai]
---

# ADR-0007: AI-first, manual-compatible development, with AGENTS.md as the operational contract

## Status

accepted

## Context

This repository is designed to operate with AI-assisted development tools (Claude Code, OpenAI
Codex CLI, Cursor, GitHub Copilot, Gemini CLI / Code Assist). Each tool has its own native
configuration mechanism (`.claude/`, `.codex/`, `.cursor/`, `.github/copilot-instructions.md`,
`.gemini/`), but without a common contract each configuration independently re-specifies the same
behavioral rules — modification scope, forbidden shortcuts, layer boundary rules, git discipline,
language requirements — creating drift risk across tools.

Agents executing on this codebase must be constrained consistently regardless of which tool or
model is active:

- Agents must not violate layer boundaries (the same rules that apply to human developers,
  enforced by CI per [ADR-0006](0006-structural-safety-via-tooling.md)).
- Agents must not modify generated files or protected configuration.
- Agents must not commit directly to protected branches.
- Agents must not introduce new architectural patterns without instruction.
- Agents must follow the OpenAPI-first flow (see ADR-0012).

Structural safety (ADR-0006) handles violations at the build layer, but behavioral constraints —
what an agent may touch, how it must behave before and after a task — cannot be expressed in a
linter. They require a declarative contract that agent harnesses load at session start. A secondary
concern is instruction priority: when the architecture doc, the rules doc, and an ad-hoc user
instruction disagree, the resolution order must be explicit.

The second question this decision settles is **which development path the repository optimizes
for**. Two facts changed the answer:

- The repository is large. Individual directories and files remain readable by a human, but
  continuously holding the whole of it — structure, conventions, dependencies, design intent, and
  the blast radius of a change — is expensive for a human working alone.
- An agent is no longer only a code generator. It is the navigation layer over that foundation:
  cross-repository search, locating the document that owns a topic, reading architecture / rules /
  skills, tracing what a change affects, assisting implementation, review, and verification, and
  collecting the friction encountered along the way.

Holding an AI-specific mechanism optional purely to keep a manual path at full parity now costs
more than the parity is worth. At the same time, the application itself must never require an AI
service to build, test, or run: a system that cannot be built, tested, or operated without a
particular vendor's agent has traded one lock-in for another.

## Decision

**The standard development path of this repository is AI-assisted. Manual development stays
technically supported, as a not-recommended compatibility path with no guarantee of an equivalent
developer experience.**

1. **AI-specific mechanisms may be built into the standard development foundation** — `.claude/`,
   `.codex/`, skills, agent-facing rules and context, AI-invoking GitHub Actions, AI review,
   feedback issues, the closed improvement loop of [ADR-0008](0008-agent-environment-alignment.md),
   skill-usage measurement, and automation that presumes an agent session. Being AI-specific is not
   by itself a reason to make a mechanism optional, and full feature symmetry with the manual path
   is not maintained.
2. **The dependency is confined to development workflow, navigation, automation, feedback, and
   review.** Application runtime, build, test, production runtime, the application architecture,
   the domain model, the API contract, the database schema, ordinary CI checks, and production
   availability stay AI-independent. With no AI service or agent available, build / test / run must
   still succeed.
3. **AI-first does not license a structure a human cannot read.** Explicit structure that an agent
   can traverse and explicit structure a human can read are treated as correlated. The existing
   responsibility separation, explicit naming, architecture boundaries, and documentation structure
   are preserved; AI-facing context, skills, and automation are added as a control surface *over*
   that structure, never as a replacement for it.
4. **Deterministic checks outrank an agent's judgment.** Tests, lint, CI, and architecture rules
   decide what an agent's reading cannot. Architecture, domain, and policy decisions keep a human
   gate.

`AGENTS.md` (project root) remains the **single operational contract** for agents, taking
precedence in the order `AGENTS.md` → `docs/rules.md` → `docs/architecture.md` → user instructions,
and remains **human-maintained only**. The contract covers instruction priority, the canonical
documentation map, the hard modification scope, forbidden shortcuts, the AI-tool configuration
directories, the skill-execution exception that relaxes scope for the duration of an explicitly
invoked skill, git rules, and language rules.

Per-tool harness files (`.claude/settings.json`, `.codex/config.toml`, …) translate and extend the
contract for each tool's native mechanism, but `AGENTS.md` remains the source of truth. Reusable
agent procedures (scaffold, review, commit, release-notes, …) are codified as skill files under
`.claude/skills/` and their `.codex/skills/` counterparts.

## Consequences

### Positive Consequences

- A single human-reviewed document governs agent behavior across all tools; per-tool configuration
  drift is reduced, and instruction priority resolves deterministically.
- The development foundation can adopt a mechanism on its merits, without first proving that a
  manual path reaches the same result by a different route.
- The line that actually carries operational risk — can this application be built, tested, and
  operated without AI — is stated once and stays testable, instead of being implied by the absence
  of AI machinery.
- The AGENTS.md and README-driven contract converges AI output variance: it raises AI-generated
  code and workflow execution toward the level a developer reaches by reading the canonical READMEs.

### Negative Consequences

- A contributor who declines the AI-assist layer performs by hand what a skill would otherwise
  drive, and encounters conventions whose only executable form is a skill. This is accepted.
- `AGENTS.md` must be kept current by humans as the architecture evolves; a stale contract misleads
  agents.
- Different AI tools have different compliance fidelity; the contract is best-effort, and hard
  enforcement remains with CI.
- The number of skills grows as the codebase matures, requiring its own maintenance discipline —
  which is why ADR-0008 puts them under a measured improvement loop rather than letting them
  accumulate.

## Alternatives Considered

### Keep the manual path as a first-class, symmetric use case

The original position: hold every AI-specific mechanism optional so that a developer without AI
tooling receives the same developer experience. Rejected. It rests on the premise that a human
alone can continuously hold the whole repository, which no longer holds at this size, and it turns
every foundation improvement into a symmetry negotiation. The genuine part of the concern — not
making the *product* depend on AI — is preserved as decision point 2 above, which is where the
operational risk actually lives.

### Drop manual development entirely

Rejected. Build, test, and runtime must not depend on an AI service, and the resulting application
must be operable without one. Removing the manual path would also make the project's viability
contingent on one vendor's agent remaining available.

### Per-tool configuration only

Each tool's native config file is the sole contract for that tool. Rejected: rules diverge across
tools and the common architectural constraints must be restated and maintained in multiple places.

### Inline AI rules in docs/rules.md

Merge agent behavioral rules into the architectural rules document. Rejected: `docs/rules.md` is an
architectural document addressed to whoever changes the code; agent-specific behavioral constraints
(modification scope, forbidden shortcuts, git discipline) reduce clarity for both audiences when
mixed in.

## Notes

- Source: `AGENTS.md` (full document — the operational contract itself).
- Skill files: `.claude/skills/` and `.codex/skills/` (per-procedure agent capabilities extending
  the contract).
- Related structural enforcement: [ADR-0006](0006-structural-safety-via-tooling.md).
- Related: [ADR-0008](0008-agent-environment-alignment.md) (how a control is judged and retired),
  [ADR-0009](0009-long-running-agent-state.md) (what agent state may persist).
- Interpretation: [`docs/design/agent-environment.md`](../design/agent-environment.md);
  `docs/architecture.md` § "AI-assisted Development".
