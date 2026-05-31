---
name: arch-check-usecase
description: Audit Go source files in `internal/usecase/**` for usecase-layer architectural compliance. Reads `CLAUDE.md` + `internal/usecase/README.md` + `internal/usecase/boundary/README.md` at runtime as the source of truth — hardcodes no rules. Runs `make lint` for depguard baseline filtered to usecase files, then performs supplementary semantic checks: forbidden imports (infrastructure packages, framework specifics — usecase must depend on domain interfaces only), thin-orchestrator heuristic (no business rules in usecase, those belong in domain entity), boundary interface usage for time / randomness / external IO (per Onion Architecture — must use `internal/usecase/boundary/`), transaction-boundary correctness (`tx.Manager` use). Confirms scope via `AskUserQuestion` (changed files vs `internal/usecase/` full). Reports violations citing source-of-truth document + line. Standalone-callable; when chained from `arch-check` integrator, receives scope context to skip its own scope question.
---

# Arch Check — Usecase

Audit `internal/usecase/**` for usecase-layer architectural compliance.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- About to commit changes to `internal/usecase/` and want layer-specific compliance check.
- Reviewing a refactor that touches Application Service orchestration.
- Standalone or chained from `arch-check` integrator.

## Source of Truth (read every run)

| Source | Purpose |
| --- | --- |
| `CLAUDE.md` (Layer Rules / Forbidden Shortcuts) | Top-level constraints |
| `internal/usecase/README.md` | Usecase responsibilities (thin orchestrator, depend on domain IF only) |
| `internal/usecase/boundary/README.md` | Boundary interface conventions (clock / tx / encrypter / external IO abstractions) |
| `.golangci.yaml` `depguard:` | Already enforced rules — do not duplicate |

## First Step: Confirm Scope

Standalone: `AskUserQuestion` with options 「変更ファイルのみ」 / 「internal/usecase/ 全体」 / 「キャンセル」. When chained from `arch-check`, skip.

## Step 1. Resolve File Scope

```sh
git diff --name-only "origin/${BASE}...HEAD" -- 'internal/usecase/**/*.go' | grep -vE '\.gen\.go$|_mock\.go$|_test\.go$'
# or
git ls-files 'internal/usecase/**/*.go' | grep -vE '\.gen\.go$|_mock\.go$|_test\.go$'
```

Empty → report and exit.

## Step 2. Lint Baseline (usecase-scoped)

```sh
make lint 2>&1 | tee /tmp/arch-check-usecase-lint.out
```

Filter to `internal/usecase/` paths. Record depguard / forbidigo violations.

## Step 3. Semantic Checks

For each usecase Go file:

1. Import scrutiny per `internal/usecase/README.md`:
   - `internal/infrastructure/**` direct imports — violation (usecase depends on domain IF, not infra impl)
   - `database/sql`, `github.com/jackc/pgx/*` — violation (DB access leak)
   - `github.com/labstack/echo` etc. — violation (framework leak)
   - `time.Now()` calls — violation (use `boundary.Clock`)
   - `math/rand`, `crypto/rand` direct — violation (use boundary if available)
2. Thin-orchestrator heuristic: function length > ~50 lines or contains conditional business-rule chains → `suggestion` "業務ロジックは domain entity へ"
3. Transaction boundary: when DB writes span multiple Repository calls, expect `tx.Manager.Do(...)` wrapping. Lack of tx wrap when multi-write → `suggestion`
4. Boundary usage: every external dependency (time, randomness, external HTTP/queue) should be injected via `internal/usecase/boundary/` interface, not used directly
5. Tracer span (recommended): per `internal/usecase/README.md` Observability section, each public method ideally starts with `ctx, endSpan := u.tracer.Start(ctx); defer endSpan()`. Missing span → `suggestion` (推奨機能、未配線は blocker ではない)

## Step 4. Report (Japanese)

```text
arch-check-usecase 結果（スコープ: <scope>）

[lint] N 件
  - <file:line>: <linter>: <message>

[semantic] M 件
  internal/usecase/user/user_usecase.go:42
    violation: "time.Now()" を直接呼び出し
    source: internal/usecase/boundary/README.md "時刻は Clock 経由"
    remediation: u.clock.Now() に置換

  internal/usecase/order/order_usecase.go:88
    suggestion: 関数 67 行、conditional 多数。業務ロジックを domain 側へ
    source: internal/usecase/README.md "Application Service は thin orchestrator"

総計: violations <N+M>, suggestions <K>
```

Empty:

```text
arch-check-usecase 結果（スコープ: <scope>）
usecase 層の違反は検出されませんでした。
```

## Step 5. Closing

- 単独実行: exit 0（情報的）
- chain: 違反数を caller に返す
- 自動修正しない

## AI Modification Scope

Read-only. No file modification.

## Constraints

- ❌ Hardcode usecase rules (always read README)
- ❌ Duplicate depguard checks
- ❌ Modify files
- ❌ Skip scope confirmation when standalone
- ✅ Japanese output
- ✅ Cite source-of-truth document + line
- ✅ Thin-orchestrator + boundary usage checks
- ✅ Re-read READMEs every run

## Checklist

- [ ] Scope confirmed or supplied
- [ ] `internal/usecase/README.md` + `boundary/README.md` read this run
- [ ] `make lint` ran, output filtered to usecase
- [ ] Generated / test files excluded
- [ ] Boundary usage + thin-orchestrator + tx checks done
- [ ] Findings cite source-of-truth
- [ ] Report Japanese
- [ ] No files modified
