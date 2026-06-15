---
name: local-review
description: Local adversarial, low-bias code review of the current change, run by subagents on a DIFFERENT model than the implementer. Mirrors `/code-review`'s finder → verify shape but keeps everything local and adds a runtime (curl + o11y) stage that mocked tests cannot cover. Confirms scope via `AskUserQuestion` (changed files vs branch-vs-base diff vs specific paths), fans out `adversarial-reviewer` subagents — one per lens (correctness / security / architecture / runtime-gap), each on `sonnet` by default so reviewer ≠ an Opus implementer — then verifies each finding with an independent `review-verifier` subagent (CONFIRMED / PLAUSIBLE / REFUTED), optionally runs the runtime curl + o11y check for touched endpoints (orchestrator-driven, per `scaffold-endpoint` Step 3.5), and synthesizes a single Japanese report. Read-only on source: reviewers cannot edit code (no fix is applied), and any destructive runtime curl is confirmed with the user first. By default the surviving CONFIRMED / PLAUSIBLE findings are posted to the branch's PR as inline review comments anchored to each finding's line (opt out with `--no-comment`; falls back to the local report when no open PR exists). Use before commit / PR to get an independent second opinion that the implementer's own model would not surface.
---

# Local Review

Independent, adversarial, **different-model** code review you can run locally — no Copilot, no cloud `/code-review`. The implementer's own model has blind spots; the whole point is to review with another model so those blind spots get caught. Built on the `/code-review` finder → verify pattern, plus a runtime curl + o11y stage that mocked unit tests structurally cannot reach.

A Japanese reference translation of this skill lives at `SKILL.ja.md` in this directory (for human reference only; not loaded as a skill).

## When to Use

- Before committing / opening a PR, to get a second opinion the implementer's model would not produce on its own.
- After a multi-layer change where mocked tests pass but DI / middleware / real-DB behavior is unverified.
- Whenever you want an adversarial pass focused on bugs, auth/IDOR, and layer violations.

Do NOT use this skill for:

- Style / formatting — `make fix` / `make lint`.
- Exhaustive layer-compliance auditing — `arch-check` (this skill's `architecture` lens flags only high-signal violations).
- Spec validation — `verify-spec`.
- Applying fixes — this skill is read-only on source; it reports, the user fixes.

## Core Idea — reviewer ≠ implementer

Bias reduction is the design constraint, not a nicety. Reviewers therefore run as **subagents on a different model than whoever wrote the code**:

- The two reviewer agents (`adversarial-reviewer`, `review-verifier`) default to **`sonnet`** in their frontmatter, which differs from the usual Opus implementer.
- **The orchestrator MUST guarantee reviewer ≠ implementer.** Check the model running this session. If it is *not* `sonnet`, the defaults are already correct — spawn as-is. If this session **is** `sonnet`, override the reviewer subagents to a different model via the `Agent` tool's `model` parameter (it takes precedence over the agent file's frontmatter) — e.g. `opus` for depth, or `haiku` for a cheap divergent pass. Never let reviewer and implementer be the same model.
- Reviewers are **read-only** (their agent files grant no Edit/Write). No fix is ever applied by this skill.

## Step 0 — Confirm Scope

Call `AskUserQuestion` immediately. Default-detect scope by checking branch vs base — get the base with `gh repo view --json defaultBranchRef -q '.defaultBranchRef.name'` (this repo's base is a `release/*` branch); if there are unmerged commits, default to "changed files", otherwise "whole working tree / specific paths".

```text
質問: どの範囲をレビューしますか？
選択肢:
  - 変更ファイルのみ（ベースブランチとの diff）  ← 未マージのコミットがある場合の既定
  - 作業ツリーの未コミット変更（git status の差分）
  - 特定のパス/ファイルを指定
  - キャンセル
```

### Flags

- `--no-comment` — suppress Step 6 (do not post to the PR); produce the local report only. **Default is opt-out**: when an open PR exists for the current branch, Step 6 posts the surviving findings as inline review comments unless this flag is given.

## Step 1 — Gather Context

- Resolve the base ref and produce the review target: `git diff <base>...HEAD` (or `git diff` for uncommitted), plus the changed-file list (`git diff --name-only ...`).
- Detect which layers/areas are touched (`internal/controller/**`, `usecase`, `domain`, `infrastructure`, `pkg`, `openapi/**`, `database/**`).
- Note whether any **endpoint** is touched (controller handler or `openapi/**`) — this decides whether Step 4 runs.
- Note whether any **shared** OpenAPI component is edited (a `components/*` referenced by more than one operation) — this widens Step 4 to every consumer.

## Step 2 — Fan-out Finders (different model, one per lens)

Spawn `adversarial-reviewer` subagents — **one per lens**, concurrently (issue all `Agent` calls in a single message). Apply the model rule from Core Idea.

| Lens | Run when |
| --- | --- |
| `correctness` | always |
| `security` | always (especially when a handler / auth / DTO / `openapi/**` is touched) |
| `architecture` | always |
| `runtime-gap` | when a controller / DI / `openapi/**` / `database/**` is touched |

Each subagent prompt MUST include: the lens name + its definition, the base ref + changed-file list + the diff, and pointers to `CLAUDE.md` / the relevant `README.md` / OpenAPI spec / migrations. Use `agentType: "adversarial-reviewer"`, `model:` per the rule, and a `label` like `find:security`.

## Step 3 — Adversarial Verify

Collect all findings and **dedup** by (file, line, claim). For each surviving finding, spawn one `review-verifier` subagent (concurrently), handing it the single finding + the base ref. Use `agentType: "review-verifier"`, `label` like `verify:<file>`.

- Keep **CONFIRMED** and **PLAUSIBLE** findings. Drop **REFUTED** (but keep a count for the report).
- For a critical/high finding where a single verdict feels shaky, spawn 2–3 verifiers and go by majority — diversity beats one opinion on the findings that matter.

## Step 4 — Runtime Verification (curl + o11y) — endpoints only

Run this **only if Step 1 found a touched endpoint**, and run it from the **orchestrator (main session)**, not a subagent — it needs interactive bash, real DB/state, log reading, and possibly user confirmation. Follow `scaffold-endpoint` Step 3.5:

1. `make test` (mocked) does NOT build the real Fx graph, run auth/OpenAPI middleware, or touch the DB — so this stage exists to catch what Step 2's `runtime-gap` lens only *predicts*.
2. Pick/seed a target row in a known state. For credential/state-sensitive checks, create a row whose plaintext/state you control.
3. `curl` the touched endpoint(s) (local auth: `Authorization: Bearer debug:<subject>`) and assert: happy path; key error paths (404 / 400 / 422); and — **if the operation declares `security:`** — no-token ⇒ 401 (prove it is actually protected). For IDOR-shaped findings, curl as a *different* subject and assert it cannot reach another subject's resource.
4. **Shared-schema impact:** if a shared `components/*` was edited (Step 1), curl **every** consumer endpoint, not just the changed one — `grep` the spec for `$ref`s and exercise each.
5. Read the o11y logs once for a single request: confirm the trace spans controller → usecase → infra and the emitted SQL is what you expect. Later re-checks can rely on o11y instead of re-curling.
6. **Destructive guard:** if a curl mutates data and the only restore path is `make db-init` (or similar), confirm with the user before running it (per `CLAUDE.md`). Clean up rows you created.

Fold any runtime-confirmed defect into the report as CONFIRMED with the curl/o11y evidence.

## Step 5 — Synthesize Report (Japanese)

Produce one Japanese report:

```text
## ローカルレビュー結果（reviewer: <model> / implementer: <model>）

スコープ: <base>...HEAD（<N> files） / lens: correctness, security, architecture, runtime-gap
ランタイム検証: 実施（curl/o11y）/ 対象外（エンドポイント変更なし）

### CONFIRMED（要対応）
- [重大度] タイトル — path:行
  - 問題 / 根拠 / 修正案
  - 検証: verifier 判定（+ 該当すれば curl/o11y 結果）

### PLAUSIBLE（要確認・判断保留）
- ...

### 補足
- REFUTED: <n> 件（finder が挙げたが verifier が否定）
- ランタイム検証でカバーした経路 / スキップした経路
```

Order by severity, CONFIRMED before PLAUSIBLE. Always state what runtime checks ran and what was skipped — silent omission reads as "covered everything" when it was not.

## Step 6 — Post Findings as Inline PR Comments (default; opt out with `--no-comment`)

By default, after Step 5, post the surviving **CONFIRMED + PLAUSIBLE** findings to the branch's PR as **inline review comments** — one per finding, anchored to its `path:line`, instead of a single wall-of-text comment. **Never post REFUTED.** The Step 5 local report is still produced regardless; this step is additive.

Skip this step entirely when:

- invoked with `--no-comment`, OR
- no open PR exists for the current branch (`gh pr view` returns nothing) — keep the local report only and optionally offer to open a PR.

Posting to GitHub is an outward-facing action, so confirm **once** before posting — show the count and the target PR (`AskUserQuestion`: 「<N> 件の指摘を PR #<番号> にインラインコメントとして投稿しますか？」/「投稿する」「投稿しない（ローカルレポートのみ）」).

### Procedure

1. Resolve PR number, repo, and the commit the comments anchor to:

   ```sh
   gh pr view --json number,url -q '.number'        # PR number
   gh repo view --json nameWithOwner -q '.nameWithOwner'
   git rev-parse HEAD                                # anchor SHA
   git rev-parse @{u}                                # pushed head — warn if it differs from HEAD
   ```

   The anchor commit MUST be the commit pushed to the PR. If local `HEAD` ≠ `@{u}`, warn the user to push first (the API rejects comments whose `commit_id` is not on the PR).

2. Decide which findings can be inline. A GitHub inline comment must target a line present in the PR diff. Parse the diff hunks (`gh pr diff <PR> --patch` or `git diff <base>...HEAD`):
   - `(path, line)` inside an added/context hunk → inline comment, `side: "RIGHT"`.
   - `(path, line)` on a removed line → inline comment, `side: "LEFT"`.
   - Off-diff (the reviewer referenced unchanged context) → **cannot** be inline; fold it into the review summary `body`.

3. Build one review and post all comments atomically (a single review, not N standalone comments):

   ```sh
   gh api --method POST repos/<owner>/<repo>/pulls/<PR>/reviews --input payload.json
   ```

   `payload.json`:

   ```json
   {
     "commit_id": "<SHA>",
     "event": "COMMENT",
     "body": "🔎 local-review (reviewer: <model>) — CONFIRMED <n> / PLAUSIBLE <m>\n\ndiff 外で行アンカー不可の指摘:\n- <path>: <要約>",
     "comments": [
       {
         "path": "<file>",
         "line": <n>,
         "side": "RIGHT",
         "body": "🔎 [CONFIRMED · high] <問題の要約>\n\n根拠: <...>\n修正案: <...>\n検証: <verifier 判定>"
       }
     ]
   }
   ```

   Use `event: "COMMENT"` — this is an advisory review, never `REQUEST_CHANGES` / `APPROVE`. Prefix every comment body with `🔎 local-review` (or the `🔎 [verdict · severity]` tag) so the posts are distinguishable from human review.

4. Robustness: if the API rejects the batch (422 — a line is not in the diff), move the offending comment(s) to the summary `body` and retry. Report afterward what was posted inline vs. summarized — never silently drop a finding.

## Do / Do NOT

- ✅ Guarantee reviewer model ≠ implementer model (override defaults if this session is sonnet).
- ✅ Run finders concurrently (one message, multiple `Agent` calls), one lens each.
- ✅ Independently verify every finding before reporting; drop REFUTED.
- ✅ Run the runtime stage for touched endpoints; widen to all consumers on a shared-schema edit.
- ✅ Confirm with the user before any destructive curl whose only restore path is `make db-init`.
- ✅ By default, post CONFIRMED + PLAUSIBLE findings to the branch's PR as inline review comments (Step 6); suppress with `--no-comment` or when no open PR exists.
- ✅ Confirm once before posting to the PR (outward action); anchor each comment to its `path:line`, fold off-diff findings into the review summary.
- ❌ Post REFUTED findings, or use `REQUEST_CHANGES` / `APPROVE` — the posted review is advisory `COMMENT` only.
- ❌ Apply fixes — this skill reports; the user fixes (reviewers are read-only by construction).
- ❌ Let a reviewer run on the same model as the implementer.
- ❌ Report speculative style nits as findings, or pad the list to look thorough.
- ❌ Edit generated files or anything in the deny list while verifying.

## Checklist

- [ ] Scope confirmed via `AskUserQuestion`; base ref resolved.
- [ ] Reviewer model verified ≠ implementer model.
- [ ] Finders fanned out per lens (concurrent).
- [ ] Every finding independently verified; REFUTED dropped (count kept).
- [ ] Runtime curl + o11y done for touched endpoints (shared-schema → all consumers); destructive curls confirmed.
- [ ] Single Japanese report: CONFIRMED → PLAUSIBLE, with runtime coverage stated.
- [ ] Unless `--no-comment` / no PR: confirmed once, then posted CONFIRMED + PLAUSIBLE as inline PR comments (off-diff → summary body); REFUTED excluded; `event: COMMENT`.
