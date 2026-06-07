---
name: arch-check-controller
description: Audit Go source files in `internal/controller/**` for controller-layer architectural compliance, with lean A convention enforcement. Reads `CLAUDE.md` + `internal/controller/README.md` + `internal/controller/handler/README.md` + OpenAPI gen `ServerInterface` at runtime as the source of truth — hardcodes no rules. Runs `make lint` for depguard baseline filtered to controller files. Lean A enforces: (1) handler method ↔ OpenAPI operationId 1:1 naming, (2) handler body = pure template (Bind → usecase call → response conversion, no business logic), (3) no Repository/infra imports in handler, (4) handler function length heuristic. No arch-check-specific annotations are introduced into implementation code — pure-template heuristic is applied uniformly and reported as `suggestion` when uncertain, leaving the user to judge. Business-logic / forbidden-import findings are `violation`. Confirms scope via `AskUserQuestion`. Standalone-callable; chained from `arch-check` integrator skips scope question.
---

# Arch Check — Controller

Audit `internal/controller/**` for controller-layer compliance, including **lean A convention enforcement** (handler body must be pure template; controller layer is derived from OpenAPI gen rather than spec-driven).

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- About to commit changes to `internal/controller/` and want layer-specific + lean A compliance check.
- Reviewing a new handler PR to confirm it follows the pure-template convention.
- Standalone or chained from `arch-check` integrator.

## Source of Truth (read every run)

| Source | Purpose |
| --- | --- |
| `CLAUDE.md` (Layer Rules / Forbidden Shortcuts) | Top-level constraints |
| `internal/controller/README.md` | Controller responsibilities |
| `internal/controller/handler/README.md` | Handler-specific conventions (pure template) |
| `internal/controller/handler/<path>/gen/server.gen.go` | OpenAPI-generated `ServerInterface` method list (for operationId ↔ handler check) |
| `.golangci.yaml` `depguard:` | Already enforced |

## Lean A Conventions (this skill enforces)

| Convention | Rationale |
| --- | --- |
| handler method 名 = OpenAPI operationId (camelCase 一致) | scaffold-controller が OpenAPI gen から導出するため |
| handler body = pure template (Bind → usecase 呼び出し → response 変換) | 業務ロジックは usecase 側、handler は HTTP I/O 変換のみ |
| 関数長 ~30 行以内（heuristic） | template 通りなら自然に収まる |
| Repository / infra package import 禁止 | layer 境界違反 |

**実装コードに arch-check 専用 annotation を導入しない方針**: pure-template チェックは body 構造から一律実施し、不確実なケースは `suggestion` 止まりにする。escape hatch が必要なら Go 標準慣習（`//nolint:<linter>` 等）に揃える検討は将来余地、独自 prefix の annotation は導入しない。

## First Step: Confirm Scope + TODO opt

Standalone: `AskUserQuestion` with 2 questions:

1. 質問: 「controller 層のどのスコープで検査しますか？」 / 選択肢: 「変更ファイルのみ」 / 「internal/controller/ 全体」 / 「キャンセル」
2. 質問: 「suggestion 検出箇所に TODO hand-off コメントを追加しますか？」 / 選択肢: 「追加する（既定）」 / 「追加しない（read-only）」

Chained from `arch-check`: skip (scope + TODO opt supplied).

## Step 1. Resolve File Scope

```sh
git diff --name-only "origin/${BASE}...HEAD" -- 'internal/controller/**/*.go' | grep -vE '\.gen\.go$|_mock\.go$|_test\.go$'
# or
git ls-files 'internal/controller/**/*.go' | grep -vE '\.gen\.go$|_mock\.go$|_test\.go$'
```

Empty → exit cleanly.

## Step 2. Lint Baseline (controller-scoped)

```sh
make lint 2>&1 | tee /tmp/arch-check-controller-lint.out
```

Filter to `internal/controller/` paths.

## Step 3. Semantic Checks (lean A enforcement)

For each handler file (`*_handler.go`):

1. **Import scrutiny** per `internal/controller/README.md`:
   - `internal/infrastructure/**` import → violation (must depend on usecase IF, not infra)
   - `internal/domain/<aggregate>/<aggregate>_repository.go` direct import → violation (use usecase)
   - DB packages → violation
2. **operationId ↔ handler method check**:
   - Find sibling `gen/server.gen.go`, extract `ServerInterface` method names
   - Confirm each method in handler file matches an `operationId` (camelCase)
   - Report extras (handler method not in `ServerInterface`) or missing (operation without handler) as `violation`
3. **Pure-template heuristic** (applied uniformly, no annotation escape):
   - Function body should match: bind to request type → call single usecase method → convert DTO to response type → return
   - Detected anti-patterns (all `suggestion` unless explicitly a hard rule):
     - Multiple `usecase.<X>` calls → 「orchestration を usecase 側に集約推奨」
     - Conditional branching beyond simple `if err != nil` → 確認推奨
     - Direct construction of domain entities → 確認推奨
     - Calls to packages outside `<handler_pkg>` / generated gen / errorhandler → 確認推奨
   - Function length > 30 lines → `suggestion`

   Hard rules (`violation`): forbidden imports (Repository / infra) — these are covered in step 1 above.
4. **Middleware / error mapping override**: if file defines custom middleware or error mapping, report as `suggestion` "endpoint 固有の特殊扱いは README に意図を記載してください"

## Step 4. TODO Hand-off Comment Insertion (opt)

If user opted into "TODO 追加" at scope confirmation (default ON), for each `suggestion`-level finding from Step 3:

1. Locate the deviation point in source (handler method line).
2. Check 3 lines immediately above for existing comment block → if found, **skip**.
3. Otherwise, insert `// TODO:` comment immediately before the handler method describing what was detected and listing resolution options for the human. Use standard `// TODO:` prefix only (no AI identifier).

The comment is a **hand-off baton**, not AI's judgment. AI does not decide whether multi-usecase orchestration is intentional or should be refactored — human resolves.

Example:

```go
// TODO: handler が複数 usecase（ListUsers, CountUsers）を呼び出している。
// orchestration を usecase 側に集約するか、本コメントを WHY 説明に置き換えてください。
func (h *Handler) GetUsers(...) {
```

`violation`-level findings (forbidden imports, operationId mismatch) do NOT get TODO comments — fix required.

## Step 5. Report (Japanese)

```text
arch-check-controller 結果（スコープ: <scope>）

[lint] N 件
  - <file:line>: <linter>: <message>

[operationId ↔ handler method] K 件
  internal/controller/handler/v1/users/v1_users_handler.go
    violation: ServerInterface に PutUsers あるが handler メソッド未実装
    source: internal/controller/handler/v1/users/gen/server.gen.go

[pure template] M 件
  internal/controller/handler/v1/orders/orders_handler.go:67
    suggestion: 関数 52 行 + 2 usecase 呼び出し
    source: internal/controller/handler/README.md "handler は pure template"
    remediation: orchestration を usecase 側に集約推奨

総計: violations <N+K+M>, suggestions <L>
TODO hand-off: 追加 <P> 件, スキップ <Q> 件（既存コメント）
```

Empty:

```text
arch-check-controller 結果（スコープ: <scope>）
controller 層の違反は検出されませんでした（lean A 規約遵守）。
```

## Step 6. Closing

- 単独: exit 0
- chain: 違反数 + TODO 追加数返却
- 自動修正なし（TODO hand-off コメントのみ追加）

## AI Modification Scope

**Mostly read-only**, with one narrow write scope:

- Reads: `CLAUDE.md` / READMEs / OpenAPI gen / in-scope Go files. Runs `make lint`.
- Writes (only with user opt at scope confirmation): `// TODO:` hand-off comments at handler-method suggestion points in `internal/controller/**/*.go`. **TODO additions only**.
- Skips comment insertion if existing comment block within 3 lines above.
- If user opts "TODO 追加なし", strictly read-only.

## Constraints

- ❌ Hardcode controller rules (always read README)
- ❌ Introduce arch-check-specific annotations into implementation code (pure-template applied uniformly, ambiguous cases are `suggestion` only)
- ❌ Modify source code (except TODO hand-off comments at suggestion points, only with user opt)
- ❌ Skip scope confirmation when standalone
- ❌ Write TODO for `violation` findings (forbidden imports / operationId mismatch → fix required)
- ❌ Decide whether multi-usecase / non-template body is intentional (hand off via TODO)
- ❌ Use AI-identifying prefix in TODO (`// TODO:` only)
- ❌ Overwrite existing comments
- ✅ Japanese output
- ✅ Cite source-of-truth + line
- ✅ lean A 規約強制（operationId ↔ method、pure template、import 禁止）
- ✅ TODO hand-off comment at suggestion points (opt-in default ON), skipped if existing comment
- ✅ Re-read READMEs + gen every run

## Checklist

- [ ] Scope + TODO opt confirmed or supplied
- [ ] CLAUDE.md / controller README / handler README / gen/server.gen.go 読込
- [ ] `make lint` filtered to controller
- [ ] operationId ↔ handler method 1:1 チェック
- [ ] pure-template heuristic (uniform application, ambiguous → suggestion)
- [ ] import 禁止リストチェック
- [ ] Findings cite source-of-truth
- [ ] Report Japanese
- [ ] TODO hand-off comments inserted at suggestion points (if opt-in), skipped where existing comment present
- [ ] No code modifications beyond TODO additions
