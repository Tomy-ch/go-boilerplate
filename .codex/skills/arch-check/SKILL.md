---
name: arch-check
description: Audit this repository's architectural compliance before review or merge. Use for changed Go files, one or more specified layers, or the full repository. Run the shared lint baseline once, inspect the applicable onion-architecture layers against their runtime READMEs, and report violations separately from advisory suggestions. Make source changes only when the user explicitly asks to add TODO hand-offs.
---

# Architecture Compliance Check

Use this skill for an architecture-focused review, not for formatting, general code review, or specification validation.

## Scope and write permission

Default to changed production Go files against the PR base or repository default branch. Exclude generated files, mocks, and tests. If there are no changed production Go files, report that and stop.

Use the full repository only when requested. A request naming one or more layers limits the audit to those layers:

| Path | Layer | Agent |
| --- | --- | --- |
| `internal/domain/` | domain | `.codex/agents/arch-auditor-domain.toml` |
| `internal/usecase/` | usecase | `.codex/agents/arch-auditor-usecase.toml` |
| `internal/controller/` | controller | `.codex/agents/arch-auditor-controller.toml` |
| `internal/infrastructure/` | infrastructure | `.codex/agents/arch-auditor-infra.toml` |
| `pkg/` | shared package | `.codex/agents/arch-auditor-pkg.toml` |

This skill is read-only by default. Do not add `TODO` comments merely because the source skill offered that option. Add them only after explicit user approval, and never add one for a violation that should be fixed.

## Procedure

1. Read `AGENTS.md`, then resolve the in-scope files and layers. Report changed Go files outside the listed layers separately; do not pretend they were audited.
2. Run `make lint` once. Attribute its findings to the appropriate layer. If the baseline is broken by unrelated errors, report the failure and do not continue semantic review.
3. For each in-scope layer, read the agent definition named in the scope table above, the layer README, and any applicable nearest package README. The agent definition gives the audit role and output contract; `AGENTS.md` and the READMEs remain the source of truth for repository rules.
4. Review layer-specific concerns:

   - **domain:** purity; inward-only dependencies; no framework, persistence, I/O, direct time/randomness/ID generation, or inappropriate context use. Treat entity-to-table correspondence as an advisory check, allowing value-object and computed-method representations.
   - **usecase:** orchestration and transaction ownership; dependency only on domain interfaces; boundary abstractions for time, randomness, and external I/O; no business invariants that belong in domain.
   - **controller:** OpenAPI operation-to-handler correspondence; request/response adaptation only; no infrastructure imports or business orchestration. Treat handler size and non-trivial branching as suggestions unless a documented rule is violated.
   - **infrastructure:** implementation of domain interfaces; data orchestration only; generated-query use and error normalization according to the RDB and `pgerror` READMEs. Treat one-to-one repository/query correspondence as advisory because joins, multi-query operations, and dispatch are valid.
   - **pkg:** no `internal/` dependency; framework-agnostic and reusable; no feature-specific business logic. Honor an explicit subpackage README exception when present.

5. Delegate independent layers to the agent roles named in the scope table above when delegation is available. Otherwise execute their instructions inline. Never run `make lint` more than once.
6. Report in Japanese using this form:

   ```text
   arch-check 結果（スコープ: <changed | full | layers>）

   [lint]
   - <file:line>: <message>

   [<layer>]
   - violation: <file:line> — <problem>
     source: <AGENTS.md or README location>
     remediation: <action>
   - suggestion: <file:line> — <advisory>
     source: <location>

   総計: violations <n>, suggestions <n>
   ```

   Do not manufacture findings. If clean, name the audited layers and say no violations were found.

## Optional TODO hand-off

Only after explicit approval, add a plain Japanese `// TODO:` comment for a suggestion in `internal/domain`, `internal/controller`, or `internal/infrastructure`.

Before insertion, inspect the three preceding lines. If a comment block already exists, skip it. Describe the observed deviation and the decision left to the human. Do not use an AI-specific marker, and do not write TODOs for violations.

## Completion constraints

- Do not auto-fix, stage, commit, or push.
- Cite the governing README or `AGENTS.md` for every finding.
- Keep generated files untouched.
- Report TODO additions and skips separately if the optional hand-off was approved.
