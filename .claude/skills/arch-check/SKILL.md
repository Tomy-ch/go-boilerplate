---
name: arch-check
description: Audit Go source files for architectural compliance against the project's onion-architecture rules. Reads `CLAUDE.md` and each touched layer's canonical README (`internal/domain/README.md`, `internal/usecase/README.md`, `internal/controller/README.md`, `internal/infrastructure/README.md`, `pkg/README.md`) as the source of truth at runtime — the skill itself hardcodes no layer rules. Runs `make lint` to surface depguard-covered violations, then performs supplementary semantic checks for rules depguard cannot express (direct stdlib/library use in domain, `pkg/` depending on `internal/`, handler bloat heuristics, layer-boundary nuances stated only in READMEs). Confirms scope with the user via `AskUserQuestion` (changed files vs full repo) before reading. Reports violations grouped by layer, each citing the source-of-truth document.
---

# Arch Check

This skill audits Go source files for architectural compliance against the project's onion-architecture rules. It treats `CLAUDE.md` and each layer's canonical README as the **source of truth at runtime** — the skill itself hardcodes no rule list. When the READMEs evolve, the skill's behavior evolves with them.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

Use this skill when:

- About to commit/push and want a layer-compliance check beyond what `make lint` covers.
- Reviewing a feature branch and want to verify it respects the onion rules end-to-end.
- After a refactor that crosses layers (e.g., moving logic between usecase and domain).
- Investigating a suspected layer violation surfaced by code review.

Do NOT use this skill for:

- Pure style / formatting issues — use `make fix` and `make lint`.
- General code review — use `/review` or `/ultrareview`.
- Discovering new files / sync drift — use `sync-readme`.

## Source of Truth (read every run)

The skill reads the following at the start of each run. It does NOT cache rules between runs.

| Source | Purpose |
| --- | --- |
| `CLAUDE.md` (sections: "Layer Rules (Strict)", "Forbidden Shortcuts", "Core Architecture") | Top-level architectural constraints |
| `internal/domain/README.md` | Domain layer rules (purity, allowed imports, value vs. interface conventions) |
| `internal/usecase/README.md` | Usecase responsibilities and boundaries |
| `internal/controller/README.md` | Controller (handler) responsibilities and forbidden patterns |
| `internal/infrastructure/README.md` | Infrastructure implementation constraints |
| `pkg/README.md` | Shared utility purity (no `internal/` deps) |
| `.golangci.yaml` (`depguard:` section) | Inventory what is already enforced by lint — do not duplicate |

For each touched layer, also read the nearest sub-package README if one exists (e.g., `internal/controller/handler/README.md`) — those often refine the parent layer's rules.

## First Step: Confirm Scope

This skill **MUST call `AskUserQuestion` immediately after invocation** to confirm scope.

Default-detect by checking:

- Current branch vs. `origin/<default>` (use `gh repo view --json defaultBranchRef -q '.defaultBranchRef.name'`) — if there are unmerged commits, propose **"changed files only"** as the default.
- If on `main` / `release/*` / no diff vs. base — propose **"full repo"** as the default.

Question text (Japanese):

- 「どのスコープでアーキ検査を実行しますか？」
- Options:
  - 「変更ファイルのみ（ベースブランチとの diff）」
  - 「指定パッケージのみ」（テキストでパス指定）
  - 「リポジトリ全体」
  - 「キャンセル」

Do NOT read any Go files or run lint before scope is confirmed.

## Step 1. Resolve Scope to a File List

Translate the chosen scope to a concrete file list.

```sh
# Changed files only
BASE=$(gh pr view --json baseRefName -q '.baseRefName' 2>/dev/null || gh repo view --json defaultBranchRef -q '.defaultBranchRef.name')
git diff --name-only "origin/${BASE}...HEAD" -- '*.go' | grep -vE '\.gen\.go$|\.sql\.go$|_mock\.go$|_test\.go$' || true

# Specific package
find <pkg-path> -name '*.go' -not -name '*.gen.go' -not -name '*.sql.go' -not -name '*_mock.go' -not -name '*_test.go'

# Full repo
git ls-files '*.go' | grep -vE '\.gen\.go$|\.sql\.go$|_mock\.go$|_test\.go$'
```

Exclusions (always):

- Generated files: `*.gen.go`, `*.sql.go`, `*_mock.go`, `**/openapi.gen.yaml`
- Vendored: `vendor/**`
- Test files (`*_test.go`) for now — testing rules are out of scope (covered by `make test` conventions documented separately)

If the resolved file list is empty, tell the user there is nothing to audit and stop.

## Step 2. Lint Baseline

Run lint and capture output:

```sh
make lint 2>&1 | tee /tmp/arch-check-lint.out
```

Parse the output for:

- `depguard` violations (these are already authoritative layer-boundary checks)
- Any other linter findings that surface architectural concerns (`forbidigo`, `gosec` etc.)

If `make lint` fails for unrelated reasons (compile error, lint config error), abort and report verbatim. Do not continue to Step 3 — semantic checks on non-compiling code are unreliable.

## Step 3. Semantic Checks (Skill-Defined)

For each Go file in scope, perform semantic checks the linters cannot express. Rules are derived **at runtime** from the source-of-truth documents read in the preceding "Source of Truth" section — do not hardcode rule lists in this skill.

For each file, determine its layer from its path:

| Path prefix | Layer |
| --- | --- |
| `internal/domain/` | domain |
| `internal/usecase/` | usecase |
| `internal/controller/` | controller |
| `internal/infrastructure/` | infrastructure |
| `pkg/` | pkg |
| (other `internal/`) | infrastructure-adjacent (cli/, system/, di/, config/, etc. — apply CLAUDE.md guidance directly) |

Then for each file:

1. Extract the `import (...)` block.
2. For each imported package, evaluate it against the rules stated in the file's layer README and CLAUDE.md.
3. Note any import that the README explicitly forbids or implicitly disallows by the layer's responsibility statement.
4. For controller files: additionally inspect handler function bodies for signs of business logic that should live in usecase (heuristic: handler function exceeds ~30 lines, contains repository-style queries, or directly performs work attributed to usecase by the controller README). Treat these as **suggestions**, not hard violations.
5. For domain files: confirm any `time` / `context` use is consistent with the domain README's stated convention (the project may permit `time.Time` as a value type while forbidding `time.Now()` calls — defer to the README's wording, not a hardcoded list).

The skill must NOT invent rules not derivable from the source-of-truth docs. If a check feels ambiguous, surface it as a "needs human judgment" item rather than a violation.

## Step 4. Report

Group findings by layer, then by file. For each finding, include:

- **Layer**: domain / usecase / controller / infrastructure / pkg
- **File:line** (line if locatable)
- **Severity**: `violation` (clear conflict with stated rule) / `suggestion` (heuristic match, may be false positive)
- **Source of truth**: which document and which line/section justifies the call
- **Suggested remediation**: a one-line hint, only if obvious; otherwise omit

Output template (Japanese):

```text
アーキテクチャ検査結果（スコープ: <scope description>）

[lint baseline]
  make lint: OK / FAIL (<n>件)
    - <violation summary if any>

[domain] <n>件
  internal/domain/foo/bar.go:12
    violation: "go.uber.org/zap" を直接 import している
    source: internal/domain/README.md L42 "domain 層は logging framework を直接利用しない"
    remediation: pkg/ または usecase 側で wrap し、interface 経由で受け取る

[controller] <n>件
  internal/controller/handler/.../baz_handler.go:88
    suggestion: handler 関数が 45 行ある（README で "lightweight" と規定）
    source: internal/controller/handler/README.md L21 "handler は軽量に保つ"
    remediation: business logic を usecase に移すことを検討

総計: violations <n>, suggestions <m>
```

If there are zero findings, report it explicitly:

```text
アーキテクチャ検査結果（スコープ: <scope description>）
違反は検出されませんでした。
```

## Step 5. Closing

- The skill does NOT auto-fix violations. Always defer to the user.
- If chained from `/commit`, exit with a non-zero status when there is at least one `violation` (not `suggestion`). When run standalone, status is informational only.
- Do not push, commit, or modify files.

## AI Modification Scope

This skill is read-only by default. It reads:

- `CLAUDE.md`, the layer READMEs, `.golangci.yaml`, all `*.go` files in scope.
- Runs `make lint` (which may write `/tmp/arch-check-lint.out`).

The skill MUST NOT:

- Modify any source file, README, or configuration.
- Stage or commit changes.
- Push to remotes.

If the user explicitly asks for an auto-fix after the report, the skill exits and recommends running a separate skill or manual edit. This skill's contract is "audit only".

## Constraints

- ❌ Hardcode layer rules in the skill — always read from READMEs / CLAUDE.md at runtime
- ❌ Duplicate checks already covered by depguard / `make lint`
- ❌ Modify any file
- ❌ Skip the scope-confirmation `AskUserQuestion`
- ❌ Treat heuristic findings (handler bloat) as hard violations
- ✅ Japanese output for user-facing messages
- ✅ Cite the source-of-truth document + line/section for each finding
- ✅ Exclude generated and vendored files
- ✅ Re-read READMEs on every invocation (no rule caching)

## Checklist

Before reporting completion, confirm:

- [ ] Scope was confirmed via `AskUserQuestion`
- [ ] CLAUDE.md and the relevant layer READMEs were read this run
- [ ] `make lint` was executed and its result was reflected in the report
- [ ] Generated and vendored files were excluded from the scope
- [ ] Each violation cites a source-of-truth document
- [ ] Heuristic findings are labeled `suggestion`, not `violation`
- [ ] No files were modified, staged, or committed
- [ ] Report is in Japanese
