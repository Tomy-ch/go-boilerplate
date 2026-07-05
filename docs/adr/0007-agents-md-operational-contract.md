---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [foundational, process, ai]
---

# ADR-0007: With-AI development — AGENTS.md as the operational contract

## Status

accepted

## Context

This repository is designed to operate safely with AI-assisted development tools (Claude
Code, Cursor, GitHub Copilot, Gemini CLI / Code Assist, OpenAI Codex CLI). Each tool is expected to have
its own native configuration mechanism (`.claude/`, `.cursor/`, `.github/copilot-instructions.md`,
`.gemini/`, `.agents/`), but without a common contract, each configuration independently
re-specifies the same behavioral rules — modification scope, forbidden shortcuts, layer
boundary rules, git discipline, language requirements — creating drift risk across tools.

AI agents executing on this codebase must be constrained consistently regardless of which
tool or model is active:

- Agents must not violate layer boundaries (the same rules that apply to human developers,
  enforced by CI per [ADR-0006](0006-structural-safety-via-tooling.md)).
- Agents must not modify generated files or protected configuration.
- Agents must not commit directly to protected branches.
- Agents must not introduce new architectural patterns without instruction.
- Agents must follow the OpenAPI-first flow (see ADR-0009).

The architecture's structural safety (ADR-0006) handles violations at the build layer, but
behavioral constraints — what an agent is allowed to touch, how it must behave before and
after a task — cannot be expressed in a linter. They require a declarative contract that
agent harnesses load at session start.

A secondary concern is instruction priority: if an agent receives conflicting guidance from
the architecture doc, the rules doc, and an ad-hoc user instruction, the resolution order
must be explicit.

## Decision

Establish `AGENTS.md` (project root) as the **single operational contract** for AI agents
interacting with this repository. It takes precedence over all other instructions in this
priority order: `AGENTS.md` → `docs/rules.md` → `docs/architecture.md` → user instructions.
`AGENTS.md` is **human-maintained only** — AI agents must not modify it.

The contract covers:

- Instruction priority order.
- Canonical documentation map (what to read before changing code).
- Hard modification scope (which directories agents may touch).
- Forbidden shortcuts (generated files, protected branches, layer bypasses).
- AI-tool configuration directories (declared out of scope for agents).
- Skill execution exception (scope relaxation during explicit skill invocations, bounded
  by the skill's own `SKILL.md` and hard-protected files).
- Git rules (branch naming, commit convention, no auto-push after PR amend).
- Language rules (all visible output in Japanese unless the user requests English).

Per-tool harness files (`.claude/settings.json`, `.cursor/rules/`, etc.) translate and
extend the contract for each tool's native mechanism, but `AGENTS.md` remains the source of
truth. Reusable agent procedures (scaffold, review, commit, release-notes, etc.) are
codified as skill files under `.claude/skills/`.

## Consequences

### Positive Consequences

- A single human-reviewed document governs AI tool behavior across all tools in the
  repository; per-tool configuration drift is reduced.
- Instruction priority is explicit: conflicts resolve deterministically.
- Protected files are enumerated (generated code, portal content, `AGENTS.md` itself),
  reducing the risk of inadvertent overwrites.
- Skill files under `.claude/skills/` codify reusable procedures without modifying the base
  contract.
- The AGENTS.md and README-driven contract also helps converge AI output variance: it raises
  AI-generated code and workflow execution toward the level a developer would achieve by
  reading the canonical READMEs.

### Negative Consequences

- `AGENTS.md` must be kept up to date by humans as the architecture evolves; a stale
  contract misleads agents.
- Different AI tools have different compliance fidelity; the contract is a best-effort
  mechanism, not a hard enforcement layer (structural enforcement remains with CI).
- The number of skill files under `.claude/skills/` grows as the codebase matures, requiring
  its own maintenance discipline.

## Alternatives Considered

### Per-tool configuration only

Each AI tool's native config file is the sole contract for that tool. Rejected: rules
diverge across tools and the common architectural constraints must be restated and maintained
in multiple places.

### Inline AI rules in docs/rules.md

Merge agent behavioral rules into the architectural rules document. Rejected:
`docs/rules.md` is an architectural document for human developers; agent-specific behavioral
constraints (modification scope, forbidden shortcuts, git discipline) reduce clarity for both
audiences when mixed in.

## Notes

- Source: `AGENTS.md` (full document — the operational contract itself).
- Skill files: `.claude/skills/` (per-procedure agent capabilities extending the contract).
- Related structural enforcement: [ADR-0006](0006-structural-safety-via-tooling.md).
- Related: `docs/architecture.md` § "AI-assisted Development".
