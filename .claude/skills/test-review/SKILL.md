---
name: test-review
description: Independent quality review of Go test files (`*_test.go`) in this repository, with adversarial finder + skeptical verifier two-stage pipeline. Defaults to `git diff` HEAD-vs-working tree to surface the changed `*_test.go` files; alternative scopes (branch-vs-base, specific paths) selectable via `AskUserQuestion`. Hardcodes no rules — reads `CLAUDE.md` Testing Instructions + the target layer's README `Test Strategy` / `Testing strategy` section + `.claude/skills/scaffold-test/SKILL.md` (the canonical generation rules) + the subject source file at runtime as the source of truth, so the reviewer stays in sync as conventions evolve (README > Code > SKILL priority). Fans out four `adversarial-reviewer` subagents on `sonnet` by default (so reviewer ≠ an Opus implementer) — one per lens: (1) **structural compliance** (`t.Parallel()` at every level / `t.Run` per subcase / outermost groups are the literal strings `正常系` / `異常系` with no `正常系_xxx` prefix form, sub-case names inside those groups carry no `正常系_` / `異常系_` prefix either / Japanese case names / `require` for errors vs `assert` for terminals per testifylint `require-error` / generated mock policy / `for`-loop usage justified / one `TestXxx` per subject); (2) **viewpoint coverage** (every sub-section in the layer README's Test Strategy is actually exercised); (3) **semantic quality** (weak assertions, brittle internals coupling, over-mocking, time-literal pinning leaks, single-`TestXxx` responsibility creep); (4) **viewpoint gap / branch × meaning completeness** (reads the subject source itself and builds a per-function two-axis matrix — Axis A 分岐網羅: every branch has a covering case; Axis B 意味網羅: each covered branch's case asserts that branch's distinctive outcome, not just that it executed — surfacing uncovered branches and covered-but-vacuously-asserted branches separately). Each surviving finding is verified by an independent `review-verifier` subagent that classifies CONFIRMED / PLAUSIBLE / REFUTED, defaulting to skepticism so plausible-but-wrong findings get filtered out. Synthesizes a single Japanese report grouped by lens with per-finding severity (修正必須 / 補完推奨 / 再考 / 追加検討). Read-only — never edits test files; the user decides what to fix and runs `scaffold-test` or hand-edits to apply. Standalone-callable; designed to slot into a PR review flow alongside `code-review` / `local-review` / `arch-check`.
---

# Test Review

Adversarial, low-bias review of Go unit test files. Read-only — surfaces what looks broken / under-tested / over-tested, and the user decides how to act.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- Before commit / PR, on the `*_test.go` files in the current change.
- After `scaffold-test` to get an independent second opinion on the generated tests.
- When coverage stays at 100 % but bug regressions still slip through (signal that test viewpoints are structurally compliant but semantically weak).
- As a standalone audit of a specific test package or file.

Do NOT use this skill for:

- Reviewing **implementation** code — use `code-review` / `local-review` / `arch-check` for that.
- Reviewing **HTTP integration tests** under `internal/integration/` — those have their own conventions documented in `internal/integration/README.md` and are better reviewed against `scaffold-integration-test`'s rules; this skill focuses on same-package unit tests.
- Applying fixes — this skill never edits files. The user runs `scaffold-test` or hand-edits afterwards.

## What This Skill Reads / Writes

**Reads (always)**:

- `CLAUDE.md` — the project-wide Testing Instructions section.
- `.claude/skills/scaffold-test/SKILL.md` — the canonical generation rules (parallel mandate, `t.Run` per subcase, 正常系 / 異常系 grouping, Japanese naming, require vs assert, mock policy, `for`-loop policy, one-`TestXxx`-per-subject policy). This skill reviews against those same rules — no duplication.
- The nearest layer README, walked up from each target test file:
  - `internal/domain/README.md` (Testing strategy)
  - `internal/usecase/README.md` (Testing Strategy)
  - `internal/controller/handler/README.md` (Test Strategy)
  - `internal/infrastructure/README.md` + `internal/infrastructure/rdb/README.md` (Test Strategy)
  - For `pkg/**`, see `scaffold-test/SKILL.md` — `pkg/README.md` intentionally has no Test Strategy section; viewpoints come from sibling tests + per-package sub-`pkg/<name>/README.md`. **No gap warning for pkg.**
- The target `*_test.go` file(s).
- The corresponding subject source file(s) (`<subject>.go` paired with `<subject>_test.go`) — required for the viewpoint-gap lens to know what is and isn't tested.
- Sibling test files in the same package — secondary reference for established conventions.

**Writes**:

- Nothing. This skill is read-only. The final output is the Japanese report rendered into the conversation.

**Triggers**: none. No `make` invocation needed (does not run the tests; it reviews them).

**Never touches**:

- Test files (no auto-fix; that is `scaffold-test` or hand-edit territory).
- Subject source files.
- Generated artifacts (`*.gen.go`, `*_mock.go`, `*.sql.go`).
- `.claude/` files.

## First Step: Resolve Scope

`AskUserQuestion`:

- Question: 「test-review の対象スコープを指定してください」
- Options (single-select):
  - 「変更ファイル (HEAD-vs-working tree, 推奨)」 — `git diff --name-only` で `*_test.go` を抽出（このフローは `local-review` / `code-review` と同じ振る舞い）。新規追加 (`git diff --name-only --diff-filter=A`) も含める。
  - 「ブランチ base 比較 (release/v1.x.y 以降の変更)」 — `git merge-base` 経由で base を解決し、その間に touched された `*_test.go`。 PR 単位で見たいとき。
  - 「特定パス / パッケージ (free-text)」 — ユーザがパスを指定。 ファイルでもディレクトリでもよい。
  - 「キャンセル」.

After resolution, build the target list. If no `*_test.go` files are in scope, stop with a friendly message — no tests to review.

For each target test file, also resolve its **subject source file** (same package, basename without `_test`). Required for the viewpoint-gap lens.

## Step 1. Read Layer Context

For every target test file:

1. Detect the layer from the file path (the band lookup matches `scaffold-test/SKILL.md`).
2. Read the layer README's `Test Strategy` / `Testing strategy` section (full text including every sub-section heading).
3. Read `CLAUDE.md` Testing Instructions once.
4. Read `.claude/skills/scaffold-test/SKILL.md` once — the canonical generation rules.
5. Read the subject source file (paired with the test file).
6. Read sibling `*_test.go` files in the same package for established conventions (helper signatures, fixture style, mock wiring).

If any layer README has no Test Strategy section in a place where one is expected (i.e. `internal/<layer>/` other than `pkg/`), note it for the report — it surfaces as a documentation gap, but does not block the review.

## Step 2. Fan Out Four Adversarial Reviewers

Spawn four `adversarial-reviewer` subagents (`subagent_type: adversarial-reviewer`) **in parallel**, each on `sonnet` by default (so reviewer ≠ an Opus implementer; the orchestrator may override the model to keep reviewer ≠ implementer).

Each subagent receives the same Step 1 context bundle (layer README, `CLAUDE.md`, `scaffold-test/SKILL.md`, target test file, subject source file, sibling tests) but a different lens prompt:

### Lens 1: Structural Compliance

Audits mechanical rule adherence — these are the hard rules surfaced by `scaffold-test/SKILL.md`:

- `t.Parallel()` is the first statement in every `t.Run` (or the block has a comment-explained `-race` exception).
- Every subcase uses `t.Run` (no inline assertions outside a `t.Run`).
- Outermost `t.Run` groups in a `TestXxx` are the literal strings `正常系` and `異常系` (each `TestXxx` may have at most one of each; finer groupings sit INSIDE those two). The `正常系_xxx` / `異常系_xxx` prefix form on the outermost group is a violation — flag it. Sub-case names inside `正常系` / `異常系` groups must NOT carry the `正常系_` / `異常系_` prefix either, since the group already labels the axis.
- Case names are Japanese.
- Error assertions use `require.*` (testifylint `require-error`); terminal value assertions use `assert.*`. `require.NotNil` / `require.True` etc. are correct ONLY when they guard subsequent code that would panic / be meaningless otherwise (e.g. `require.NotNil(fn); fn(...)` or `require.NotNil(rec); _ = rec.Field`); when nothing uses the value afterwards the check is terminal and must be `assert.*` — flag a terminal `require.NotNil` / `require.True` / `require.Equal` as a violation.
- A subtest that drives a generated mock asserts via the mock controller: `EXPECT()` expectations — including a deliberate *no-EXPECT* (or `.Times(0)`) to assert a method is never called — ARE the assertion. Do NOT flag such a subtest as "assertion-less" merely because it has no `assert.*` / `require.*` line.
- `for`-loop / table-driven blocks have an obvious readability justification; otherwise sequential `t.Run` is expected.
- **1:1 mapping between subject functions and `TestXxx`.** The *subject* is the paired production source — the non-test, non-generated `.go` file that the binary is built from; `*.gen.go` / `*.sql.go` / `*_mock.go` and test-only helpers are out of scope (no hand-written `TestXxx` expected). Check BOTH directions:
  - *forward*: each `TestXxx` covers exactly one subject function / method — a `TestXxx` bundling multiple subjects needs the one-line rationale comment required by `scaffold-test/SKILL.md`.
  - *reverse*: each subject function / method maps to exactly one `TestXxx`. A subject split across multiple `TestXxx` (e.g. `TestFoo` + `TestFoo_Metrics` + `TestFoo_CloseError`, or a `Test_foo` / `TestFoo_foo` naming-variant pair) is a finding → consolidate into a single `TestXxx` whose `正常系` / `異常系` groups absorb the variants (Rule 7). A public subject function with NO `TestXxx` at all is a coverage gap (its branches also surface in Lens 4 Axis A).
- Mocks come from `<package>/mock/*_mock.go` — no hand-written mocks.
- No imports of `internal/` from `pkg/**` test files; no infrastructure imports from `internal/domain/**` test files; etc. (architectural rules in `CLAUDE.md`).

Output: a structured finding list with `file:line` references and the violated rule.

### Lens 2: Viewpoint Coverage

Compares the layer README's Test Strategy sub-sections to what the test file actually exercises:

- For each sub-section heading in the README's Test Strategy (`### Getter contract test` / `### Immutable guarantee test` / `### Invariant preservation test` / etc.), is there at least one `TestXxx → t.Run(case)` that maps to it?
- If a heading is absent from the README (e.g. `pkg/**`), use the sibling-test pattern as the comparison baseline instead.
- Specific examples to look for, derived from the current READMEs (descriptive, not hardcoded — when READMEs change, the reviewer adapts):
  - domain: pointer immutability tests (per `Immutable guarantee test` and `TestImmutableAccessors`), invariant preservation across state transitions, VO boundary checks.
  - usecase: orchestration mock-call order, transaction-boundary application, boundary call usage.
  - controller: HTTP I/O conversion, validation, apperror → status mapping, middleware-supplied context (auth principal / request id).
  - infra: SQL execution paths, `pgerror.NormalizeError` application, row → entity conversion.

Output: a list of viewpoints the README declares but the test file does not exercise, with `file:section` references back into the README.

### Lens 3: Semantic Quality

Audits whether the assertions are actually meaningful:

- **Weak assertions**: `assert.NotNil(t, x)` as the only check for a complex return value; `assert.NoError` without follow-up state assertions; `assert.Equal(t, len(actual), 1)` instead of asserting on the element.
- **Name over-promising the assertion**: a `t.Run` case name claims a property the body does not actually verify — e.g. `"…を保持した収集器を返す"` while the body only asserts `NotNil`, when the held value lives in an unexported field or is another unit's responsibility. Either assert the distinctive property or rename the case to what is verified. Corollary: for a **branchless pass-through / wiring function** (e.g. a DI provider that just forwards its input to a constructor), one honest case is correct — extra cases that re-run the same `NotNil` assertion with different inputs add no coverage (no branch distinguishes them) and should be collapsed.
- **Brittle internals coupling**: tests reading unexported fields when the public API would do; tests asserting on logging output or error message *strings* without `errors.Is`.
- **Over-mocking**: every collaborator mocked when a real (pure) implementation would be lighter and more revealing; mock setup verifying call counts at a granularity that locks the implementation in place.
- **Time-literal pinning leaks**: `time.Now()` called inside the assertion rather than a fixed `baseTime`; comparisons relying on system clock.
- **`TestXxx` responsibility creep**: one `TestXxx` driving multiple subjects without a recorded rationale (rule violation already, but also a semantic smell when the rationale is weak).
- **Helper duplication**: a 5+-line fixture repeated across three `TestXxx` functions that should be a `t.Helper()`-tagged helper.
- **Redundant comments**: inline comments that restate the code or narrate *why* (rather than behavior). The project keeps test comments minimal — case intent lives in the Japanese `t.Run` name, not in comments. Flag restated-identifier comments and test-rationale narration left in the test body (one-line godoc-style declaration comments are exempt; the `-race` serial-block exception comment is required, not redundant).

Output: a list of findings with `file:line` and a one-sentence explanation of why the assertion is weak or brittle.

### Lens 4: Viewpoint Gap — Branch × Meaning Completeness (subject-driven)

Reads the subject source file itself and builds, **per function / method**, a two-axis completeness matrix. Code coverage ≠ meaningful coverage: a branch can be executed by a case that asserts nothing about what makes that branch distinct, and isolating that gap is the point of this lens. Run both axes for every subject — a branch is only "done" when it is both reached (Axis A) and distinctly asserted (Axis B).

**Axis A — branch enumeration (分岐網羅)**: every logical branch in the subject is reached by at least one case.

- Every conditional branch (positive / negative) has at least one `t.Run` case.
- Every error sentinel (`ErrInvalid*` / `apperror.*`) declared or returned is reached by at least one case.
- Every boundary value pair (min-1 / min / max / max+1) for a constrained field is exercised if the subject enforces it.
- Constructor / factory functions that defend against zero-value / nil input have a rejecting case.
- A branch reached only by *executing* a constructor / provider / `fx.Invoke` body that the test's harness never runs is still uncovered. Notably `fx.ValidateApp` validates the dependency graph WITHOUT executing constructors or lifecycle hooks — so a graph-validation test does NOT cover a provider / invoke body; that needs a direct unit test (call the function) for branch coverage.
- A `t.Skip` whose reason claims the branch "cannot be reproduced" is itself an Axis-A gap to challenge, not to accept: check whether an integration-style harness reaches it (e.g. real-DB lock contention / `55P03` needs two independent connections + transactions because a serialized test-tx helper cannot represent concurrency). Surface the skipped branch as 追加検討 with the concrete reproduction path.

A branch with NO covering case is a **分岐未カバー** finding → severity **追加検討** (proactive). Cite the subject `file:line` of the uncovered branch + a proposed `t.Run` case name.

**Axis B — meaning coverage (意味網羅)**: each covered branch's case actually asserts that branch's *distinctive* outcome, not merely that it executed.

- An error branch asserts the specific sentinel via `require.ErrorIs` — not just `require.Error`.
- A success branch asserts the resulting value / state that distinguishes it from the other branches — not just `require.NoError` / `assert.NotNil`.
- A state-mutating method's case asserts the post-mutation invariant / changed field — not just that the call returned.
- A pointer-returning getter using `ptr.Copy` has an immutability assertion, not just a value-equality check.
- A boundary case asserts the differing outcome on each side of the boundary (accept vs reject), not just the accept side.

A branch that IS covered but whose case does not distinctly assert its outcome is a **分岐カバー済み・意味未検証** finding → severity **再考** (it passes and lifts coverage but reveals nothing). Tie the finding to the specific subject branch + the test case that nominally covers it.

Output: per subject, (1) uncovered branches with proposed `t.Run` case names (Axis A → 追加検討), and (2) covered-but-vacuously-asserted branches with the missing distinctive assertion (Axis B → 再考), each citing the subject branch `file:line` and the covering test case.

## Step 3. Verify Each Finding

Each surviving finding goes to an independent `review-verifier` subagent (`subagent_type: review-verifier`) on `sonnet`. The verifier:

- Re-derives the conclusion from the code itself, **not** trusting the finder.
- Defaults to skepticism — when ambiguity remains, label PLAUSIBLE or REFUTED rather than CONFIRMED.
- Classifies the finding as **CONFIRMED** (the rule violation / gap / weak assertion is real and reproducible) / **PLAUSIBLE** (looks right but verifier could not fully reproduce the chain of reasoning) / **REFUTED** (verifier finds counter-evidence — e.g. the cited line is a comment, the heading is from a different layer, the assertion is actually sufficient given context).

Verification runs in parallel across findings (`parallel(findings.map(f => () => agent(verifyPrompt(f))))`). The orchestrator may keep the verifier on a different model than the finder where useful.

REFUTED findings are dropped from the synthesized report (but a count is mentioned in the summary so the user knows the noise floor). CONFIRMED and PLAUSIBLE findings are kept and grouped by lens for the synthesis.

## Step 4. Synthesize Report

Produce a single Japanese report. Recommended structure:

```text
# Test Review レポート

対象: <スコープ + ファイル一覧>
レンズ: 構造準拠 / 観点カバレッジ / 意味的品質 / 観点ギャップ
verifier 通過: CONFIRMED <n> 件 / PLAUSIBLE <m> 件 / REFUTED <k> 件 (フィルタ済み)

## サマリ
- 修正必須: <件数>
- 補完推奨: <件数>
- 再考: <件数>
- 追加検討: <件数>

## 構造準拠（修正必須）
- [<severity>] <file>:<line> — <violated rule>
  - 詳細: <description>
  - 出典: `<README path>` / `scaffold-test/SKILL.md` の該当節
  - verifier: CONFIRMED / PLAUSIBLE

## 観点カバレッジ（補完推奨）
- [<severity>] <test file> — <missing viewpoint>
  - 出典 README: `<README path>:<section heading>`
  - 提案: <suggested t.Run case name>
  - verifier: CONFIRMED / PLAUSIBLE

## 意味的品質（再考）
- [<severity>] <file>:<line> — <weak assertion / brittle coupling>
  - 詳細: <one-sentence why>
  - verifier: CONFIRMED / PLAUSIBLE

## 観点ギャップ: 分岐網羅（追加検討）
- <file> に対して subject <subject path> から導出:
  - 分岐未カバー: <subject file:line の分岐>
  - 提案: t.Run("<case name>", ...) — カバーする分岐 / sentinel: <reason>
  - verifier: CONFIRMED / PLAUSIBLE

## 観点ギャップ: 意味網羅（再考）
- <file> に対して subject <subject path> から導出:
  - 分岐カバー済み・意味未検証: <subject file:line の分岐> を <test file:line のケース> がカバーするが固有 outcome 未 assert
  - 不足アサーション: <あるべき distinctive assertion>
  - verifier: CONFIRMED / PLAUSIBLE

## 補遺
- pkg/ 層は Test Strategy 節を意図的に持たないため、sibling tests を比較基準にしています（gap 警告なし）。
- <その他、レビュー過程で気付いた README 補完候補 / SKILL の改訂候補>
```

Severity mapping:

- **修正必須** (Structural Compliance lens): rule violations against `CLAUDE.md` / `scaffold-test/SKILL.md` — these are hard rules. CONFIRMED → 修正必須; PLAUSIBLE → 確認推奨.
- **補完推奨** (Viewpoint Coverage lens): README declares a viewpoint that is not exercised. CONFIRMED → 補完推奨; PLAUSIBLE → 確認推奨.
- **再考** (Semantic Quality lens + Viewpoint Gap Axis B): the test compiles and passes but reveals little — a weak assertion, or a branch that is covered yet does not distinctly assert its outcome. CONFIRMED → 再考; PLAUSIBLE → 補強候補.
- **追加検討** (Viewpoint Gap Axis A): proactive suggestion for an uncovered branch found by subject inspection. CONFIRMED → 追加検討; PLAUSIBLE → 提案.

## Step 5. Next-Action Suggestion

End the report with a single concrete suggestion:

- If 「修正必須」 findings exist → suggest re-running `/scaffold-test` on the affected subjects (which will reset the structure to the canonical pattern) or hand-editing the specific lines.
- If only 「補完推奨」 / 「追加検討」 findings exist → suggest adding the proposed `t.Run` cases either manually or via a follow-up `/scaffold-test` invocation with those subjects in scope.
- If no findings survive verification → state that explicitly (`「verifier 通過後 0 件です」`).

## Chainability

This skill is designed to stand alongside `code-review` / `local-review` / `arch-check` in a PR review flow. It does not currently chain *into* other skills — the user reads the report and decides whether to invoke `scaffold-test` (regenerate) or hand-edit. A future PR-review orchestrator could fan out `code-review` + `arch-check` + `test-review` in parallel and merge their reports; that orchestrator does not exist yet.

When chained from such an orchestrator in the future, the parent passes a context payload with at least:

- `scope` — pre-resolved file list (skips First Step's `AskUserQuestion`).
- `base_ref` — when running in branch-vs-base mode.
- `skip_verifier` — boolean; allows the parent to disable the verify stage for speed (default `false`, i.e. verify by default).

## Constraints (Summary)

- ❌ Editing any file (read-only).
- ❌ Running `make test` (this skill reviews tests, not runs them; coverage / pass-status is `make test`'s job, run separately).
- ❌ Trusting finder output without verification (the verifier stage is mandatory unless the parent passes `skip_verifier: true`).
- ❌ Hardcoding viewpoint lists (the SSOT is the layer README's Test Strategy section; `pkg/` is the documented exception).
- ❌ Duplicating rules already in `CLAUDE.md` or `scaffold-test/SKILL.md` (the skill reads them at runtime).
- ✅ Default to skepticism in the verifier (PLAUSIBLE > CONFIRMED when ambiguous).
- ✅ Default reviewer model is `sonnet` (different from Opus implementers); orchestrator may override to keep reviewer ≠ implementer.
- ✅ Default scope is changed files; alternative scopes selectable.
- ✅ Final report in Japanese, grouped by lens with severity tags.
- ✅ Surface `pkg/` as an intentional "no Test Strategy" layer, never as a documentation gap.

## Checklist

Before reporting completion, confirm:

- [ ] Scope was resolved (changed files / base diff / explicit paths).
- [ ] Each target `*_test.go` has its subject source file located.
- [ ] Layer README + `CLAUDE.md` + `scaffold-test/SKILL.md` + sibling tests were read in Step 1.
- [ ] All four lenses ran (in parallel).
- [ ] Lens 4 ran both axes per subject — Axis A 分岐網羅 (uncovered branches → 追加検討) and Axis B 意味網羅 (covered-but-vacuously-asserted branches → 再考).
- [ ] Every finding from every lens went through `review-verifier` (unless `skip_verifier: true`).
- [ ] REFUTED findings were dropped; CONFIRMED / PLAUSIBLE were kept.
- [ ] Final report is Japanese, grouped by lens, with severity tags.
- [ ] Next-action suggestion is one concrete recommendation.
- [ ] No files were edited.
