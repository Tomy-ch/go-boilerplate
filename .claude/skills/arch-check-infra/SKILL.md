---
name: arch-check-infra
description: Audit Go source files in `internal/infrastructure/**` for infrastructure-layer architectural compliance, with lean A convention enforcement. Reads `CLAUDE.md` + `internal/infrastructure/README.md` + `internal/infrastructure/rdb/README.md` + sqlc gen (`internal/infrastructure/rdb/sqlc/gen/*.gen.go`) at runtime as the source of truth — hardcodes no rules. Runs `make lint` for depguard baseline filtered to infrastructure files. Lean A enforces (with soft 1:1 — multi-query / switch dispatch / JOIN are legitimate): (1) Repository method dispatches to 1+ sqlc gen functions (multi-call allowed for joins / aggregations / N+1 resolution; auto-inferred from body inspection), (2) Repository body = data orchestration only (sqlc calls + row→entity conversion + `pgerror.NormalizeError`, **no business logic**), (3) Repository must implement a domain Repository IF, (4) `pgerror.NormalizeError` use on all sqlc returns. No arch-check-specific annotations are introduced into implementation code — legitimate multi-query / dispatch patterns are recognized from body structure. Business-logic detection in Repository body is `violation`; missing sqlc-gen correspondence is `suggestion`. Confirms scope via `AskUserQuestion`. Standalone-callable; chained from `arch-check` integrator skips scope question.
---

# Arch Check — Infra

Audit `internal/infrastructure/**` for infrastructure-layer compliance, including **lean A convention enforcement** (Repository body must be pure template; infra layer is derived from domain IF + sqlc gen rather than spec-driven).

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- About to commit changes to `internal/infrastructure/` and want layer + lean A check.
- Reviewing a new Repository implementation PR.
- Standalone or chained from `arch-check` integrator.

## Source of Truth (read every run)

| Source | Purpose |
| --- | --- |
| `CLAUDE.md` (Layer Rules) | Top-level constraints |
| `internal/infrastructure/README.md` | Infrastructure layer rules |
| `internal/infrastructure/rdb/README.md` | RDB layer specific conventions |
| `internal/infrastructure/rdb/pgerror/README.md` | Error normalization conventions |
| `internal/infrastructure/rdb/sqlc/gen/*.gen.{sql.go,go}` | Sqlc-generated function list (`*.gen.sql.go` for query files, `*.gen.go` for db / models) — both used for Repository ↔ sqlc gen check |
| `internal/domain/<aggregate>/<aggregate>_repository.go` | Domain Repository IF (for impl-coverage check) |
| `.golangci.yaml` `depguard:` | Already enforced |

## Lean A Conventions (this skill enforces)

| Convention | Rationale |
| --- | --- |
| Repository method が 1+ の sqlc gen 関数を呼ぶ（多重 OK、switch dispatch OK、JOIN 用複数 query OK） | scaffold-infra-db は 1:1 mechanical case のみ自動導出、それ以外は hand-write |
| Repository body = データ orchestration のみ（sqlc 呼び出し + pgerror + 行→entity 変換）。**業務ロジック厳禁** | 業務ロジックは domain entity / usecase の責務 |
| `pgerror.NormalizeError` が全 sqlc return で呼ばれる | DB エラーの統一正規化 |
| 全 Repository が対応する domain Repository IF を実装 | layer 境界 |

**実装コードに arch-check 専用 annotation を導入しない方針**: 多重 sqlc 呼び出し / switch dispatch / JOIN は body 構造から自動推論。escape hatch が必要なら Go 標準慣習（`//nolint:<linter>` 等）に揃える検討は将来余地として残すが、独自 prefix の annotation 系は導入しない。実装コードはあくまで実装の主体であり、補助ツールに引きずられない。

## First Step: Confirm Scope + TODO opt

Standalone: `AskUserQuestion` with 2 questions:

1. 質問: 「infrastructure 層のどのスコープで検査しますか？」 / 選択肢: 「変更ファイルのみ」 / 「internal/infrastructure/ 全体」 / 「キャンセル」
2. 質問: 「suggestion 検出箇所に TODO hand-off コメントを追加しますか？」 / 選択肢: 「追加する（既定）」 / 「追加しない（read-only）」

Chained from `arch-check`: skip.

## Step 1. Resolve File Scope

```sh
git diff --name-only "origin/${BASE}...HEAD" -- 'internal/infrastructure/**/*.go' | grep -vE '\.gen\.go$|_mock\.go$|_test\.go$'
git ls-files 'internal/infrastructure/**/*.go' | grep -vE '\.gen\.go$|_mock\.go$|_test\.go$'
```

Empty → exit.

## Step 2. Lint Baseline (infra-scoped)

```sh
make lint 2>&1 | tee /tmp/arch-check-infra-lint.out
```

Filter to `internal/infrastructure/` paths.

## Step 3. Semantic Checks (lean A enforcement)

For each repository file (`internal/infrastructure/rdb/repository/**/*_repository.go`):

1. **Import scrutiny**:
   - direct `database/sql` import → suggestion (use sqlc gen wrapper)
   - missing `internal/infrastructure/rdb/pgerror` import → likely violation if Repository methods exist
   - direct framework imports unrelated to DB → violation
2. **Domain IF coverage**:
   - Find matching `internal/domain/<aggregate>/<aggregate>_repository.go`
   - Confirm Repository struct implements all IF methods (compile-time enforced; skill double-checks IF method names ↔ struct method names)
3. **Repository method ↔ sqlc gen check (soft)**:
   - For each Repository method, find at least one corresponding function call to `internal/infrastructure/rdb/sqlc/gen/*.gen.go`
   - **Allow legitimate multi-query patterns**:
     - Multiple sqlc gen calls combined (JOIN, N+1 解決, aggregation+detail) — body 内の `sqlc.*` 呼び出し回数で検出、自動許容
     - Switch dispatch across multiple sqlc gens（パラメータ駆動、例: `FindByActive`）
   - **Report as `suggestion`** (not `violation`):
     - Repository method calls no function in `sqlc/gen/` → 「sqlc gen 呼び出しが見当たりません。実装見直しを推奨」
     - Method name doesn't match any sqlc gen function name (heuristic match by stem) → 確認推奨

4. **Body composition heuristic**:
   - Expected shape: tracer span → 1+ sqlc gen calls → row→entity conversion → return（`pgerror.NormalizeError` 経由）
   - **`violation` (hard rule)**:
     - Business logic detected in body — examples: entity invariant validation, domain-level decision branching, calculations on data values beyond simple field copy / nil-coalesce / type cast → 「業務ロジックは domain / usecase の責務」
     - sqlc 呼び出し戻り値の error を `pgerror.NormalizeError` 経由せず raw で返す
   - **`suggestion`**:
     - 関数長 50 行超（switch dispatch 除く） → 複雑度の確認
     - tracer span 未配線
5. **Observability**:
   - `tracer.Start` call missing in Repository method → `suggestion` (per `internal/infrastructure/README.md` observability section)

For query_service / system_query directories, apply the same rules (they share the pure-template + sqlc wrap convention).

## Step 4. TODO Hand-off Comment Insertion (opt)

If user opted into "TODO 追加" at scope confirmation (default ON), for each `suggestion`-level finding from Step 3 (Repository ↔ sqlc gen soft mismatch / function-length suggestion / tracer span missing):

1. Locate the Repository method line.
2. Check 3 lines immediately above for existing comment block → if found, **skip**.
3. Otherwise, insert `// TODO:` comment immediately before the method describing what was detected and listing resolution options. Standard `// TODO:` prefix only.

The comment is a **hand-off baton**. AI does NOT decide whether multi-query / dispatch is intentional — human resolves.

Example:

```go
// TODO: FindUserWithPosts は users / posts の 2 つの sqlc gen を組み合わせている。
// 意図通りなら本コメントを WHY 説明に置き換えてください。
func (r *userRepository) FindUserWithPosts(...) {
```

`violation`-level findings (business logic in body / missing `pgerror.NormalizeError`) do NOT get TODO comments — fix required.

## Step 5. Report (Japanese)

```text
arch-check-infra 結果（スコープ: <scope>）

[lint] N 件
  - <file:line>: <linter>: <message>

[Repository ↔ sqlc gen] K 件（suggestion only）
  internal/infrastructure/rdb/repository/user/user_repository.go
    suggestion: Save メソッドが sqlc gen 関数を呼んでいません
    remediation: 対応 query 追加 or 実装の見直し

[body composition] M 件
  internal/infrastructure/rdb/repository/order/order_repository.go:42
    violation: NormalizeError 経由なし、errors.New(...) で直接返している
    source: internal/infrastructure/rdb/pgerror/README.md "全 sqlc エラーは NormalizeError 経由"

  internal/infrastructure/rdb/repository/order/order_repository.go:60
    violation: Repository body に業務ロジック検出（amount > 0 のような entity 不変条件チェック）
    source: internal/infrastructure/README.md "Repository は data orchestration のみ"
    remediation: invariant check を domain entity 側へ

総計: violations <M>, suggestions <K+...>
TODO hand-off: 追加 <P> 件, スキップ <Q> 件（既存コメント）
```

Empty:

```text
arch-check-infra 結果（スコープ: <scope>）
infrastructure 層の違反は検出されませんでした（lean A 規約遵守）。
```

## Step 6. Closing

- 単独: exit 0
- chain: 違反数 + TODO 追加数返却
- 自動修正なし（TODO hand-off コメントのみ追加）

## AI Modification Scope

**Mostly read-only**, with one narrow write scope:

- Reads: `CLAUDE.md` / READMEs / sqlc gen / domain Repository IF / in-scope Go files. Runs `make lint`.
- Writes (only with user opt at scope confirmation): `// TODO:` hand-off comments at Repository-method suggestion points in `internal/infrastructure/**/*.go`. **TODO additions only**.
- Skips comment insertion if existing comment block within 3 lines above.
- If user opts "TODO 追加なし", strictly read-only.

## Constraints

- ❌ Hardcode infra rules
- ❌ Treat sqlc gen 1:1 不一致を hard violation 扱い（suggestion のみ）
- ❌ Introduce arch-check-specific annotations into implementation code (multi-query / dispatch are inferred from body structure)
- ❌ Modify source code (except TODO hand-off comments at suggestion points, only with user opt)
- ❌ Skip scope confirmation when standalone
- ❌ Write TODO for `violation` findings (business logic / missing pgerror → fix required)
- ❌ Decide whether multi-query / dispatch is intentional (hand off via TODO)
- ❌ Use AI-identifying prefix in TODO (`// TODO:` only)
- ❌ Overwrite existing comments
- ✅ Japanese output
- ✅ Cite source-of-truth + line
- ✅ lean A 強制（Repository ↔ sqlc gen soft、body 業務ロジック禁止 strict、pgerror strict、IF impl coverage）
- ✅ multi-query / switch dispatch / JOIN を body 構造から自動推論（実装コードに annotation 不要）
- ✅ TODO hand-off comment at suggestion points (opt-in default ON), skipped if existing comment
- ✅ Re-read READMEs + sqlc gen + domain IF every run

## Checklist

- [ ] Scope + TODO opt confirmed or supplied
- [ ] CLAUDE.md / infra README / rdb README / sqlc gen / domain Repository IF 読込
- [ ] `make lint` filtered to infra
- [ ] Repository ↔ sqlc gen 1:1 check
- [ ] body composition heuristic (multi-query / dispatch auto-inferred from body)
- [ ] `pgerror.NormalizeError` use check
- [ ] Tracer span check
- [ ] Findings cite source-of-truth
- [ ] Report Japanese
- [ ] TODO hand-off comments inserted at suggestion points (if opt-in), skipped where existing comment present
- [ ] No code modifications beyond TODO additions
