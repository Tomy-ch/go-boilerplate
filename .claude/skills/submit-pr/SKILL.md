---
name: submit-pr
description: Push the current feature branch to `origin` and create or update its GitHub pull request. Detects whether a PR already exists for the current branch via `gh pr view` and automatically chooses between "create" and "update". The PR body is filled from `.github/pull_request_template.md` (sections `概要` / `変更内容` / `動作確認方法`) using the commit history and diff. Title and body are written in Japanese per `CLAUDE.md`. The skill confirms with the user before any push, with the exact wording required by `CLAUDE.md` for the update path.
---

# Submit PR

This skill pushes the current branch to `origin` and ensures a GitHub pull request exists for it. It handles two cases automatically:

- **Create**: no PR exists for the current branch → push (with `-u` if no upstream) and open a new PR.
- **Update**: an open PR already exists → confirm with the user, then push (the PR's diff auto-updates).

The PR body is filled from `.github/pull_request_template.md`. The skill never auto-pushes, never overwrites an existing PR's title/body, and never force-pushes.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## Preconditions

- `gh` CLI is installed and authenticated (`gh auth status` succeeds).
- Current branch is not a protected branch (`production` / `develop` / `staging` / `release/*`).
- Working tree is clean. If there are uncommitted changes, the skill aborts and suggests running `/commit` first.

## Step 0. Pre-flight Checks

Run in parallel:

```sh
git rev-parse --abbrev-ref HEAD                          # current branch
git status --porcelain                                   # working-tree state
git rev-parse --verify '@{u}' 2>/dev/null                # upstream existence
git log '@{u}'..HEAD --oneline 2>/dev/null               # unpushed commits (if upstream)
gh auth status
```

Bail out if any of the following:

- Branch matches `^(production|develop|staging|release/.+)$` → tell the user to switch to a feature branch.
- `git status --porcelain` is non-empty → tell the user to run `/commit` (or stash) first.
- `gh auth status` fails → tell the user to run `gh auth login`.

The four valid working states going into Step 1:

| Upstream | Unpushed commits | Meaning |
| --- | --- | --- |
| none | n/a | First push case |
| set | > 0 | Subsequent push case |
| set | 0 | Nothing to push; PR may still need to be created |
| set | 0 + PR open | Nothing to do (handled in Step 1) |

## Step 1. Detect Existing PR and Base Branch

```sh
gh pr view --json number,state,baseRefName,headRefName,url,title,body 2>/dev/null
gh repo view --json defaultBranchRef -q '.defaultBranchRef.name'
```

Branch on the result:

- **PR exists and state is `OPEN`** → "update" path. Base branch is fixed (`baseRefName` from the result).
- **PR exists but state is `MERGED` / `CLOSED`** → ask the user via `AskUserQuestion`:
  - Question: 「このブランチには `<state>` 状態の PR #N があります。新規 PR を作成しますか？」
  - Options: 「新規 PR を作成する」 / 「キャンセル」
- **No PR exists** → "create" path. Base branch defaults to the repo's default branch.

For the "create" path, if multiple `release/*` branches exist locally and the user may want a non-default target, confirm via `AskUserQuestion`:

- Question: 「ベースブランチをこれで作成しますか？」
- Options: 「`<default-branch>` を使う」 / 「別のブランチを指定する」

Special early-exit cases:

- "update" path with 0 unpushed commits → tell the user there is nothing to push and stop. Print the existing PR URL.
- "create" path with 0 unpushed commits but the remote branch exists → continue to Step 2 (we will create a PR for whatever is already on the remote).

## Step 2. Gather Context and Read Template

Collect the inputs needed to compose title and body. `<base>` is the base branch decided in Step 1.

```sh
git log <base>..HEAD --pretty=format:'%h %s'                # commit titles
git log <base>..HEAD --pretty=format:'%h%n%s%n%b%n---'      # commit titles + bodies
git diff <base>...HEAD --shortstat                          # diff summary
git diff <base>...HEAD --name-only                          # changed files
```

Read `.github/pull_request_template.md` and identify sections by `#` / `##` headers. The current template defines:

- `# 概要`
- `## 変更内容`
- `## 動作確認方法`

Strip the HTML comment placeholders. If the template is absent, fall back to the same three-section structure inline.

## Step 3. Compose Title and Body

### Title

- Derive from the most significant change. Single-commit PR → use that commit's title (strip the leading `<Prefix>:` only if redundant). Multi-commit PR → summarize the overall intent in Japanese.
- ≤ 70 characters.
- If the branch name embeds an issue number (`feature/1234-...`, `bugfix/5678-...`), include `#1234` in the title naturally.
- For the "update" path: keep the existing PR title unchanged unless the user explicitly asks to change it.

### Body

Fill each template section in Japanese:

- **概要**: 1–3 sentences summarizing the PR's intent. Use commit messages as the primary source.
- **変更内容**: Bullet list grouped by area (API / DB / 内部ロジック / テスト / ドキュメント など). Reference changed files and commit titles. Group meaningfully — do not paste a raw file list.
- **動作確認方法**: Concrete verification steps. Adapt to what actually changed: `make serve` + curl for API changes, `make db-local-migrate-up` for migrations, `make test` for logic, etc.

If the branch name encodes an issue number, append `closes #N` at the bottom of the body (or fold it into 概要 if natural).

## Step 4. Confirm with the User

Before the push confirmation, surface a **non-blocking** reminder that an independent, different-model review is recommended before the change leaves the local machine:

> 推奨: push 前に `/local-review`（実装者とは別モデルの独立・敵対的レビュー）を実行しましたか？ 未実行なら一度回すと、モックテストでは出ない不具合（認証/IDOR・DI/SQL・共有スキーマ波及など）を PR 前に拾えます。

This is a recommendation only — never block the push on it, and never auto-run the review. If the user has already reviewed or declines, continue.

Display the resolved title, base branch, push command, and full body.

### Create path

`AskUserQuestion`:

- Question: 「以下の内容で PR を作成しますか？」
- Options:
  - 「この内容で作成する」
  - 「draft で作成する」
  - 「title / body を修正したい」
  - 「キャンセル」

If the user chooses "修正したい", collect free-text feedback, regenerate the relevant section, and re-confirm.

### Update path

Display the unpushed commit list and diff summary. Then ask with the wording required by `CLAUDE.md`:

- Question: 「変更はローカルにコミット済みです。これらの変更をプルリクエストにプッシュしますか？」
- Options: 「push する」 / 「キャンセル」

## Step 5. Push

```sh
# First push (no upstream)
git push -u origin <branch>

# Subsequent push
git push
```

Never use `--force` or `--force-with-lease` unless the user has explicitly requested it.

On push failure (non-fast-forward, permission denied, network error, etc.), report the error verbatim to the user and stop. Do not attempt automatic recovery.

## Step 6. Create or Update the PR

### Create the PR

```sh
gh pr create \
  --base "<base-branch>" \
  --title "<title>" \
  --body "$(cat <<'EOF'
<body>
EOF
)" [--draft]
```

### Update the PR

Step 5's push already updated the PR's diff. Do NOT touch the PR's title or body by default.

Only if the user explicitly asked to update them, run:

```sh
gh pr edit <number> [--title "<new-title>"] [--body "$(cat <<'EOF'
<new-body>
EOF
)"]
```

## Step 7. Report

Print the PR URL and a brief summary in Japanese.

For the create path:

```text
PR を作成しました: <url>
ベース: <base-branch>
タイトル: <title>
コミット数: N
```

For the update path:

```text
PR を更新しました: <url>
追加コミット数: N
```

## Step 8. Post-PR Review (confirm)

After the PR URL is reported, **always ask the user whether to run a review** — do not skip this, and do not auto-run a review. Use `AskUserQuestion`:

- Question: 「PR を作成/更新しました。コードレビューを実行しますか？」
- Options (offer the ones that apply):
  - 「`/local-review` を実行」 — local diff, different-model adversarial review (strong on auth / IDOR / DI / SQL / shared-schema gaps that mocked tests miss)
  - 「`/code-review <PR#>` を実行」 — PR-based review (can post inline comments with `--comment`)
  - 「ultrareview を案内」 — cloud multi-agent review; **user-triggered and billed**, so the skill cannot launch it — only surface the command for the user to run
  - 「レビューしない」

### Depth by change type

Scale the recommended depth to what changed (if a full Go / JS source review is 10):

- **Behavior-affecting code** (`internal/**`, `pkg/**` `.go`, SQL, OpenAPI) → full depth (10); recommend a review by default.
- **Docs / tooling-dominant changes** (`docs/**`, `*.md`, `.claude/**`, `AGENTS.md`, CI config — no production behavior change) → shallower is acceptable (~7–8/10). Still ask, but note the lower ROI so the user can decide quickly.

Judge the dominant nature of the diff (changed file paths / commit prefixes) to pick the default recommendation, but the user's choice always wins.

## Constraints

- ❌ Push to protected branches (`production` / `develop` / `staging` / `release/*`)
- ❌ `git push --force` / `--force-with-lease` (only with explicit user instruction)
- ❌ Auto-update an existing PR's title or body (only on explicit user request)
- ❌ Push while the working tree has uncommitted changes
- ❌ Create a PR without user confirmation
- ❌ Push to an existing PR branch without re-confirming with the exact wording required by `CLAUDE.md`
- ✅ Use `.github/pull_request_template.md` as the body skeleton
- ✅ Japanese title and body
- ✅ HEREDOC for the body when calling `gh pr create` / `gh pr edit`
- ✅ Detect issue number from branch name and surface it in title / body

## Checklist

Before reporting completion, confirm:

- [ ] Current branch is not a protected branch
- [ ] Working tree was clean before the push
- [ ] `gh auth status` passed
- [ ] PR template was read and reflected in the body
- [ ] Title and body are Japanese
- [ ] Title ≤ 70 characters
- [ ] (推奨) push 前に `/local-review` の実行を確認した（非ブロッキング）
- [ ] User confirmation was obtained before the push (mandatory for update path per `CLAUDE.md`)
- [ ] PR URL was reported to the user
- [ ] (必須) PR 作成/更新後にレビュー実行可否を確認した（深さは変更種別でスケール）
- [ ] No `--force` was used
