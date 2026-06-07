---
name: arch-check-domain
description: Audit Go source files in `internal/domain/**` for domain-layer architectural compliance. Reads `CLAUDE.md` + `internal/domain/README.md` at runtime as the source of truth — hardcodes no rules. Runs `make lint` for depguard baseline filtered to domain files, then performs supplementary semantic checks: forbidden imports (logging frameworks, context.Context per README rules, time.Now() calls, infra-only packages), entity field ↔ SQL migrations column **soft correspondence** (lean A — `database/migrations/*.sql` `CREATE TABLE` columns vs entity struct fields). The skill auto-infers legitimate divergence patterns from code structure (computed values expressed as methods rather than fields, VO types that wrap multiple SQL columns). No source-code annotations are introduced — implementation code stays free of arch-check metadata. Mismatches are reported as `suggestion`, not `violation`, since 1:1 is an idealization (calculated fields, VO groupings, input splits, audit-derived values are legitimate). Confirms scope via `AskUserQuestion`. Standalone-callable; chained from `arch-check` integrator skips scope question.
---

# Arch Check — Domain

Audit `internal/domain/**` for domain-layer architectural compliance.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- About to commit changes to `internal/domain/` and want layer-specific compliance check.
- Reviewing a refactor that touches domain entities or Repository interfaces.
- Standalone or chained from `arch-check` integrator.

Do NOT use for:

- Other layers — use the matching `arch-check-<layer>` skill.
- Pure style / formatting — `make fix` / `make lint`.

## Source of Truth (read every run)

| Source | Purpose |
| --- | --- |
| `CLAUDE.md` (Layer Rules / Forbidden Shortcuts) | Top-level constraints |
| `internal/domain/README.md` | Domain layer rules (purity, allowed imports, value vs interface conventions) |
| `internal/domain/<aggregate>/*.go` (existing aggregate package) | Reference patterns from sibling code (per-aggregate READMEs are intentionally not used — principles live in the top-level domain README) |
| `database/migrations/*.sql` | SQL `CREATE TABLE` for entity ↔ column check (lean A) |
| `.golangci.yaml` `depguard:` | Already enforced rules — do not duplicate |

## First Step: Confirm Scope + TODO opt

If invoked standalone, `AskUserQuestion` with 2 questions:

1. 質問: 「domain 層のどのスコープで検査しますか？」 / 選択肢: 「変更ファイルのみ」 / 「internal/domain/ 全体」 / 「キャンセル」
2. 質問: 「suggestion 検出箇所に TODO hand-off コメントを追加しますか？」 / 選択肢: 「追加する（既定）」 / 「追加しない（read-only）」

When chained from `arch-check` integrator with scope + TODO opt already supplied, skip these.

## Step 1. Resolve File Scope

```sh
# Changed files only (filter to domain)
BASE=$(gh pr view --json baseRefName -q '.baseRefName' 2>/dev/null || gh repo view --json defaultBranchRef -q '.defaultBranchRef.name')
git diff --name-only "origin/${BASE}...HEAD" -- 'internal/domain/**/*.go' | grep -vE '\.gen\.go$|_mock\.go$|_test\.go$' || true

# Full domain
git ls-files 'internal/domain/**/*.go' | grep -vE '\.gen\.go$|_mock\.go$|_test\.go$'
```

Empty → report and exit cleanly.

## Step 2. Lint Baseline (domain-scoped)

```sh
make lint 2>&1 | tee /tmp/arch-check-domain-lint.out
```

Filter output to `internal/domain/` paths. Capture depguard / forbidigo / gosec findings as `violation` severity.

If lint itself fails for unrelated reasons, abort with verbatim output.

## Step 3. Semantic Checks

For each domain Go file in scope:

1. Extract `import (...)` block.
2. Cross-reference each import against `internal/domain/README.md`'s allowed / forbidden lists. Common forbidden patterns:
   - `go.uber.org/zap`, `go.uber.org/fx` — logging / DI frameworks
   - `github.com/labstack/echo`, `database/sql`, `gorm`, etc. — framework / infra
   - `time.Now()` / `uuid.New()` calls — per `internal/domain/README.md` "Handling time and ID": time and ID generation must be done outside Domain (allow `time.Time` and `uuid.UUID` as value types)
   - `context.Context`: **allowed only in Repository interface method signatures** (per README Repository examples). Any other use (e.g., domain entity behavior method, value object factory) → `suggestion`. CLAUDE.md broadly forbids context.Context in Domain, but the README defines a Repository-IF carve-out — README wins per the README > Code > SKILL priority
3. For entity files (struct defining an Aggregate Root), cross-check with the matching SQL migration as a **soft check** — **no source-code annotations are introduced**; legitimate divergences are auto-inferred from code structure:
   - Find `database/migrations/*.sql` containing `CREATE TABLE <aggregate_plural>` (use the latest migration that defines the table; later ones add/alter columns)
   - Map `snake_case` columns ↔ `camelCase` struct fields
   - **Auto-recognized legitimate divergences (no finding)**:
     - Computed values expressed as methods (e.g., `func (u User) FullName() string`) — not fields, so naturally excluded from the field check
     - VO types that wrap multiple SQL columns (e.g., a `Money` field whose type contains `Currency` + `Amount`) — skill resolves the VO type and treats wrapped columns as covered
     - Method-only structs (no struct fields at all)
   - **Report as `suggestion`** (not `violation`) when:
     - SQL column without any matching struct field or VO-wrapped equivalent → 「永続化されないカラムです。entity 追加か migration 削除を検討」
     - Struct field with no matching column and no VO resolution → 「SQL に対応カラムなし。計算値ならメソッド形式に書き換え、不要なら削除を検討」
     - Type mismatch (e.g., `VARCHAR` vs `int`) → 確認推奨

   **Convention (not annotation)**: 計算値は **メソッド形式**で書くのが既存規約。これにより struct field 検査の対象外となり、annotation 不要で正当な逸脱を表現できる。

   理由: 1:1 strict は ideal model で、現実には計算値 / VO 群化 / 入力分解 / audit 由来値等の正当な逸脱がある。検出は人間レビューの起点とし、blocker にはしない。実装コードに arch-check 専用 annotation を入れない（補助ツールはコードを読んで自動推論する設計）。

## Step 4. TODO Hand-off Comment Insertion (opt)

If the user opted into "TODO 追加" at scope confirmation (default ON), for each `suggestion`-level finding from Step 3:

1. Locate the deviation point in source (struct field line / file).
2. Check the 3 lines immediately above the deviation for an existing comment block. If found → **skip** (don't duplicate).
3. Otherwise, write a `// TODO:` comment immediately before the deviation point that:
   - States what was detected (e.g., "struct field has no matching SQL column")
   - Lists resolution options for the human (migration, method form, in-memory + WHY, etc.)
   - Uses standard `// TODO:` prefix only (no `// TODO(arch-check):` or other AI identifier)

The comment is a **hand-off baton**, not AI's judgment. AI does NOT decide whether the deviation is intentional or a bug — the human resolves by either fixing the code or replacing the TODO with a WHY comment.

Example:

```go
// TODO: User 構造体に phoneNumber フィールドあり、SQL カラム未定義。
// 永続化が必要なら migration 追加、計算値ならメソッド形式に書き換え、
// in-memory 保持なら本コメントを WHY 説明に置き換えてください。
phoneNumber string
```

`violation`-level findings do NOT get TODO comments (those should be fixed, not deferred).

## Step 5. Report (Japanese)

```text
arch-check-domain 結果（スコープ: <scope>）

[lint] N 件
  - <file:line>: <linter>: <message>

[semantic] M 件
  internal/domain/foo/foo.go:12
    violation: "go.uber.org/zap" を直接 import している
    source: internal/domain/README.md L42 "domain 層は logging framework を直接利用しない"

[entity ↔ SQL] K 件（suggestion only）
  internal/domain/user/user.go vs database/migrations/0003_users.sql
    suggestion: User 構造体に `phoneNumber` フィールドあり、SQL カラム未定義
    remediation: 計算値ならメソッド形式に書き換え、永続化必要なら migration 追加、VO ラップで複数カラム束ねるなら型変更を検討

総計: violations <N+M>, suggestions <K+...>
TODO hand-off: 追加 <P> 件, スキップ <Q> 件（既存コメント）
```

Empty result:

```text
arch-check-domain 結果（スコープ: <scope>）
domain 層の違反は検出されませんでした。
```

## Step 6. Closing

- 単独実行: レポート出力して exit。violations があっても exit 0（情報的）
- `arch-check` から chain: 違反数 + TODO 追加数を caller に返す（integrator が集約報告）
- 自動修正しない（コード本体は触らず、TODO hand-off コメントのみ追加）

## AI Modification Scope

**Mostly read-only**, with one narrow write scope:

- Reads: `CLAUDE.md`, READMEs, `database/migrations/*.sql`, in-scope Go files. Runs `make lint` (may write `/tmp/*.out`).
- Writes (only with user opt at scope confirmation): `// TODO:` hand-off comments at deviation points in `internal/domain/**/*.go`. **Only TODO additions** — no code changes, no `violation` auto-fix, no existing comment modification.
- Skips comment insertion if any comment block exists within 3 lines above the deviation (de-dup).
- If user opts "TODO 追加なし" at scope, strictly read-only.

## Constraints

- ❌ Hardcode domain rules (always read `internal/domain/README.md`)
- ❌ Duplicate depguard checks
- ❌ Modify source code (except TODO hand-off comments at deviation points, only with user opt)
- ❌ Skip scope confirmation when standalone
- ❌ Write TODO comments for `violation`-level findings (those should be fixed, not deferred)
- ❌ Decide whether a deviation is intentional ("AI が解決済み判定しない" — hand off to human via TODO)
- ❌ Use AI-identifying prefix in TODO comments (`// TODO:` only, no `// TODO(arch-check):` or similar)
- ❌ Overwrite or modify existing comments at deviation points
- ✅ Japanese output
- ✅ Cite source-of-truth document + line
- ✅ Entity ↔ SQL soft check (lean A; suggestion only; method-form values and VO wrapping auto-recognized as legitimate divergences)
- ✅ TODO hand-off comment at suggestion points (opt-in default ON), skipped if existing comment present
- ✅ Re-read READMEs every run

## Checklist

- [ ] Scope + TODO opt confirmed (standalone) or supplied (chained)
- [ ] `CLAUDE.md` + `internal/domain/README.md` read this run
- [ ] `make lint` ran, output filtered to domain
- [ ] Generated / test files excluded
- [ ] Entity ↔ SQL soft check executed for entity files (suggestion only, with escape hatches respected)
- [ ] TODO hand-off comments inserted at suggestion points (if opt-in), skipped where existing comment present
- [ ] Findings cite source-of-truth document
- [ ] Report is Japanese
- [ ] No code modifications beyond TODO additions
