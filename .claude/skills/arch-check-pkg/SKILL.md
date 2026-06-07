---
name: arch-check-pkg
description: Audit Go source files in `pkg/**` for shared-utility purity rules. Reads `CLAUDE.md` + `pkg/README.md` + nearest sub-`pkg/<name>/README.md` at runtime as the source of truth — hardcodes no rules. Runs `make lint` for depguard baseline filtered to pkg files. Enforces: (1) `pkg/` must NOT depend on `internal/` (top-level CLAUDE.md rule — pkg is reusable across layers), (2) framework-agnostic (no echo / fx / gorm / etc. unless explicitly allowed by sub-package), (3) no business logic specific to a single feature. Confirms scope via `AskUserQuestion` (changed files vs `pkg/` full). Reports violations citing source-of-truth document + line. Standalone-callable; chained from `arch-check` integrator skips scope question.
---

# Arch Check — Pkg

Audit `pkg/**` for shared-utility purity rules.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- About to commit changes to `pkg/` and want layer-specific compliance check.
- Reviewing a new utility addition to `pkg/`.
- Standalone or chained from `arch-check` integrator.

## Source of Truth (read every run)

| Source | Purpose |
| --- | --- |
| `CLAUDE.md` ("pkg/" section) | Top-level constraint: no `internal/` deps, framework-agnostic |
| `pkg/README.md` | pkg layer rules |
| `pkg/<name>/README.md` (nearest sub-README if exists) | Sub-package specific refinements |
| `.golangci.yaml` `depguard:` | Already enforced |

## Key Rules (derived from CLAUDE.md / README)

| Rule | Source |
| --- | --- |
| `pkg/` must NOT import `internal/**` | CLAUDE.md "pkg must not depend on infrastructure or framework-specific packages" |
| framework-agnostic (no echo / fx / gorm 等 unless explicitly noted) | CLAUDE.md / pkg/README.md |
| no feature-specific business logic | pkg/README.md |

## First Step: Confirm Scope

Standalone: `AskUserQuestion` 「変更ファイルのみ」 / 「pkg/ 全体」 / 「キャンセル」. Chained: skip.

## Step 1. Resolve File Scope

```sh
git diff --name-only "origin/${BASE}...HEAD" -- 'pkg/**/*.go' | grep -vE '\.gen\.go$|_mock\.go$|_test\.go$'
git ls-files 'pkg/**/*.go' | grep -vE '\.gen\.go$|_mock\.go$|_test\.go$'
```

Empty → exit.

## Step 2. Lint Baseline (pkg-scoped)

```sh
make lint 2>&1 | tee /tmp/arch-check-pkg-lint.out
```

Filter to `pkg/` paths.

## Step 3. Semantic Checks

For each pkg Go file:

1. **`internal/` import check** (the cardinal rule):
   - Any `<module>/internal/**` import → `violation`
   - Source: CLAUDE.md "pkg/" section
2. **Framework import scrutiny** per `pkg/README.md`:
   - `github.com/labstack/echo`, `go.uber.org/fx`, ORM packages, etc. → `violation` unless sub-package README explicitly allows
3. **Feature-specific logic heuristic**:
   - Function / type name references a specific aggregate (e.g., `UserSomething`) → `suggestion` "feature 固有のロジックは `internal/` 側へ"

## Step 4. Report (Japanese)

```text
arch-check-pkg 結果（スコープ: <scope>）

[lint] N 件
  - <file:line>: <linter>: <message>

[internal/ 依存] K 件
  pkg/foo/foo.go:5
    violation: "go-boilerplate/internal/domain/user" を import している
    source: CLAUDE.md "pkg must not depend on infrastructure or framework-specific packages"

[framework 依存] M 件
  pkg/bar/bar.go:8
    violation: "github.com/labstack/echo" を import している
    source: pkg/README.md "framework-agnostic"

総計: violations <N+K+M>, suggestions <L>
```

Empty:

```text
arch-check-pkg 結果（スコープ: <scope>）
pkg 層の違反は検出されませんでした。
```

## Step 5. Closing

- 単独: exit 0
- chain: 違反数返却
- 自動修正なし

## AI Modification Scope

Read-only.

## Constraints

- ❌ Hardcode pkg rules
- ❌ Modify files
- ❌ Skip scope confirmation when standalone
- ✅ Japanese output
- ✅ Cite source-of-truth + line
- ✅ `internal/` 依存禁止 + framework 非依存チェック
- ✅ Re-read READMEs every run

## Checklist

- [ ] Scope confirmed or supplied
- [ ] CLAUDE.md + pkg/README.md + sub-README 読込
- [ ] `make lint` filtered to pkg
- [ ] `internal/` import チェック
- [ ] framework import スクリプチェック
- [ ] Findings cite source-of-truth
- [ ] Report Japanese
- [ ] No files modified
