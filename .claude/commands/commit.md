---
description: Analyze the current working-tree changes (staged + unstaged), group them into appropriately-scoped commits with the project's prefix convention (Feat / Fix / Refactor / Perf / Docs / Test / Build / CI / Chore / Style / Revert), and execute each commit in Japanese after user approval. Commits are made with `git commit --no-verify` to skip lefthook; the user is responsible for running lefthook (or equivalent checks) manually before pushing. Respects CLAUDE.md's git rules (no direct commits to protected branches, no force-push, no auto-push after PR amend, Co-Authored-By footer, HEREDOC commit messages).
argument-hint: [--dry-run] [--scope=staged|all]
allowed-tools: Bash(git status:*), Bash(git diff:*), Bash(git log:*), Bash(git add:*), Bash(git commit:*), Bash(git branch:*), Bash(git rev-parse:*), Bash(git reset:*), Bash(make fix:*), Read, AskUserQuestion
---

# Commit

You have been invoked via `/commit`. Argument string: `$ARGUMENTS`

This command analyzes uncommitted changes in the working tree and produces one or more git commits with appropriate granularity and the project's prefix convention. All commit messages are in Japanese, per `CLAUDE.md`.

This command intentionally bypasses lefthook on every commit (`git commit --no-verify`) so that pre-commit checks (`make lint` / `make test` / `make sql-lint` / migration checks) do not fire N times during multi-commit splits. Verification is the user's responsibility — they should run `lefthook run pre-commit` (or `make lint` / `make test`) manually before pushing.

## Step 0. Auto-format

Run `make fix` once at the very start to absorb formatting fixes (gofmt / goimports / auto-fixable lint rules). This removes the most common source of noise from the subsequent diff inspection and reduces the chance the user's later lefthook check fails on pure formatting.

```sh
make fix
```

If `make fix` itself fails, abort and report the failure to the user. Do not continue. Any changes it produces are folded into the working tree and become part of the candidate change set inspected in Step 2.

## Step 1. Pre-flight Checks

Run these in parallel:

```sh
git rev-parse --abbrev-ref HEAD                      # current branch
git rev-parse HEAD                                   # current HEAD commit (save as ORIGINAL_HEAD)
git status --porcelain                               # staged + unstaged
git diff --shortstat                                 # unstaged summary
git diff --staged --shortstat                        # staged summary
git rev-parse --verify MERGE_HEAD 2>/dev/null        # detect ongoing merge
git rev-parse --verify CHERRY_PICK_HEAD 2>/dev/null  # detect ongoing cherry-pick
git rev-parse --verify REBASE_HEAD 2>/dev/null       # detect ongoing rebase
```

Save the current HEAD commit hash as `ORIGINAL_HEAD`. This is the rollback target if anything fails during Step 5.

Bail out (do not commit) if any of the following:

- Current branch matches `^(production|develop|staging|release/.+)$`. Per `CLAUDE.md` git rules, never commit to protected branches. Inform the user and ask them to create a feature branch first (e.g., `feature/<issue-or-topic>`).
- Both staged and unstaged porcelain outputs are empty. Tell the user there is nothing to commit and stop.
- Any of `MERGE_HEAD` / `CHERRY_PICK_HEAD` / `REBASE_HEAD` is set. The repository is mid-operation; ask the user to resolve that first.

Read `.lefthook.yaml` (if present) and extract the list of `pre-commit:` command entries. The list is displayed in Step 4 so the user knows what is being skipped. If `.lefthook.yaml` is absent, note that and continue.

Parse `$ARGUMENTS`:

| Flag | Effect |
| --- | --- |
| `--dry-run` | Produce the grouping proposal but do not stage or commit. |
| `--scope=staged` | Only consider currently-staged changes. |
| `--scope=all` | Consider both staged and unstaged (default). |

## Step 2. Inspect Changes

Collect detailed diffs to understand the nature of each change:

```sh
git diff --staged                     # full staged diff
git diff                              # full unstaged diff
git diff --staged --name-only
git diff --name-only
```

Treat the following as **rider files** — they never form their own commit, but ride along with the source change that produced them:

- Generated files: `**/*.gen.go`, `**/*.sql.go`, `*_mock.go`, `**/openapi.gen.yaml`, generated content under `docs/portal/links/implements/`
- Vendored content: `vendor/**`

Example: an `openapi/**/*.yaml` change brings its `*.gen.go` outputs with it in the same commit. A `database/dml/**/*.sql` change brings its `internal/infrastructure/rdb/sqlc/gen/*.gen.go` outputs in the same commit.

## Step 3. Prefix Reference

Use exactly **one** of the following prefixes per commit (capitalized, English, colon-suffixed):

| Prefix | Purpose | Examples |
| --- | --- | --- |
| `Feat:` | New feature, new endpoint, new migration | New handler, new API in `openapi/`, new SQL under `database/migrations/` |
| `Fix:` | Bug fix (correcting behavior that deviates from intent) | Error-handling fix, logic correction |
| `Refactor:` | Internal cleanup without changing external behavior | Function split, rename, responsibility move, layer reorganization |
| `Perf:` | Performance improvement | Query optimization, N+1 elimination, allocation reduction |
| `Docs:` | Documentation change | `README*`, `docs/`, `*.ja.md`, code comments, release notes |
| `Test:` | Adding or fixing tests | `*_test.go`, test fixtures, test helpers |
| `Build:` | Build system, dependencies, tooling | `Dockerfile`, `go.mod` / `go.sum`, `Makefile`, `.makefiles/**`, `tools.yaml` |
| `CI:` | CI/CD configuration | `.github/workflows/**`, `lefthook.yml`, GitHub Actions related |
| `Chore:` | Miscellaneous chores | `.gitignore`, editor settings, `.claude/**`, other small tasks |
| `Style:` | Formatting-only changes that do not affect logic | Output of `make fix`, `gofmt`, `goimports` |
| `Revert:` | Undoing an existing commit | Output of `git revert`, or an equivalent manual revert |

Do not invent prefixes outside this list. When ambiguous, choose the closest match (most cases are one of `Feat` / `Fix` / `Refactor`).

### Path-based hints

| Path pattern | Candidate prefix |
| --- | --- |
| `internal/**/*.go` (non-test) | `Feat` / `Fix` / `Refactor` / `Perf` (judge from the diff) |
| `**/*_test.go` | `Test` |
| `openapi/**/*.yaml` | `Feat` (API change) |
| `database/migrations/**/*.sql` | `Feat` (schema change) |
| `database/dml/**/*.sql` | `Feat` / `Refactor` (new query vs. cleanup) |
| `docs/**/*.md`, `README*.md`, `*.ja.md` | `Docs` |
| `Dockerfile`, `docker/**`, `go.mod`, `go.sum`, `Makefile`, `.makefiles/**`, `tools.yaml` | `Build` |
| `.github/workflows/**`, `lefthook.yml`, `.lefthook.yaml` | `CI` |
| `.gitignore`, `.claude/**`, editor settings | `Chore` |

## Step 4. Propose Grouping

Build a list of proposed commits with appropriate granularity. Each item:

```txt
[N] <Prefix>: <short Japanese title>
    files:
      - path/to/file1
      - path/to/file2
    rationale: <why these belong in one commit>
```

### Granularity guidance

- **One semantic change = one commit.** Do not mix feature + refactor + fix into a single commit.
- **Tests may co-locate with the implementation they cover** (a new handler and its tests belong together). If you are only adding tests for existing code, that goes into a standalone `Test:` commit.
- **Generated artifacts co-locate with their source change.** When `openapi/*.yaml` changes, the regenerated `*.gen.go` files belong in the same commit. The output of `make gen-api` / `make gen-query` follows the same rule.
- **Formatting-only changes are standalone `Style:` commits.** Output produced by Step 0's `make fix` may be folded into the appropriate existing group when it is clearly part of the same change; if it is unrelated, surface it as a separate `Style:` commit.
- **`Docs:` is standalone by default.** Exception: when documentation is part of a new feature (e.g., a README added alongside a new package), they may co-locate.
- **One prefix per commit.** If you feel the urge to write two, the grouping is wrong.

### Lefthook notice

Along with the grouping proposal, display the lefthook commands that will be **skipped** during this `/commit` run. Read them dynamically from `.lefthook.yaml` (the list is configuration, not hardcoded). Example output when the current config defines lint/test/sql-lint/migration checks:

```txt
This command will run `git commit --no-verify` on every commit.
The following lefthook pre-commit commands will be SKIPPED:
  - lint                    (make lint)
  - test                    (make test)
  - sql-lint                (make sql-lint)
  - migration-check-version (make check-migration-up-version check-migration-down-version)
  - migration-check-gap     (make check-migration-up-gap check-migration-down-gap)
Run `lefthook run pre-commit` (or `make lint` / `make test`) manually before pushing.
```

### Confirmation

Confirm the proposal via `AskUserQuestion`:

- Question: 「提案したコミット分割でよいですか？」
- Options: 「この提案で進める」 / 「修正したい箇所を指摘する」

When `--dry-run` is set, print the proposal and stop. Do not stage or commit.

## Step 5. Execute Each Commit

For each approved group, run the following in order:

```sh
# Stage only the files belonging to this group (never use -A / .)
git add path/to/file1 path/to/file2

# HEREDOC is required (preserves the title / blank line / body / footer layout).
# --no-verify is intentional: lefthook is bypassed by design (see Step 4 notice).
git commit --no-verify -m "$(cat <<'EOF'
<Prefix>: <short Japanese title>

<Optional body: what changed and why>

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

### Commit message rules

- **Title**: `<Prefix>: <Japanese title>`, aim for 50 characters or fewer.
- **Body**: Optional. If present, leave one blank line after the title and wrap around 72 characters. Prefer "why" over "what".
- **Language**: Japanese (per the output rule in `CLAUDE.md`).
- **`Co-Authored-By` footer**: Required. Use `Claude Opus 4.7 (1M context) <noreply@anthropic.com>`.
- **HEREDOC**: Required (keeps the title + blank line + body + footer layout intact).
- **`--no-verify`**: Required for every commit produced by this command. This is an explicit, command-scoped carve-out from the project-wide rule; the rationale is documented in Step 4 (lefthook is run once manually before push, not N times during the split).
- **Never use `-a`, `git add -A`, or `git add .`.** Always stage files by name (avoids sweeping in `.env` or credentials).
- **`--no-gpg-sign` and `--amend` remain prohibited.**

### Error handling

If `git add` or `git commit` fails for any group (file-path typo, mid-operation state that slipped through pre-flight, GPG signing failure, etc.):

1. Stop further commits immediately. Do not continue with the next group.
2. Report to the user:
   - Which group failed (`[k]` index and proposed title)
   - The captured stderr from the failed command
   - The commits already created in this session: `git log --oneline <ORIGINAL_HEAD>..HEAD`
3. Use `AskUserQuestion` to ask how to recover:
   - Question: 「ここまでに作成したコミットをどうしますか？」
   - Options:
     - 「ロールバックする (`git reset --mixed <ORIGINAL_HEAD>`)」 — rewinds HEAD to the saved `ORIGINAL_HEAD`, leaves all changes in the working tree, clears the index
     - 「そのまま残して停止する」 — keep the partial commits and hand control back to the user
4. If the user chooses rollback, run `git reset --mixed <ORIGINAL_HEAD>` and confirm with `git status` and `git log --oneline -n 3`. Never use `--hard`.

## Step 6. Push Policy and Final Reminder

- **Do not auto-push** (per `CLAUDE.md` git rules).
- After all commits succeed, report to the user using this template:

  ```txt
  N 件のコミットを作成しました（lefthook はスキップしています）。
  push 前に以下のいずれかを実行して動作確認してください:
    - lefthook run pre-commit
    - make lint && make test
  プッシュは手動で実行してください: `git push`
  ```

- When working on an existing PR branch, follow `CLAUDE.md` and ask before pushing:
  「変更はローカルにコミット済みです。これらの変更をプルリクエストにプッシュしますか？」

## Constraints (Summary)

- ❌ Direct commits to `production` / `develop` / `staging` / `release/*` branches
- ❌ Auto-running `git push` / `git push --force` / `git reset --hard` / `git checkout --` / `git clean -f`
- ❌ `--no-gpg-sign` / `--amend`
- ❌ `git add -A` / `git add .` / `git commit -a` (always name files explicitly)
- ❌ Mixing multiple prefixes in one commit
- ❌ Committing without `--no-verify` (would run lefthook N times)
- ✅ Japanese commit messages
- ✅ HEREDOC for the message
- ✅ `Co-Authored-By` footer
- ✅ `--no-verify` on every commit produced by this command
- ✅ Stage only the files in the current group
- ✅ `make fix` once at Step 0 before inspection
- ✅ Capture `ORIGINAL_HEAD` at Step 1 for safe rollback
- ✅ On failure, propose `git reset --mixed <ORIGINAL_HEAD>` via `AskUserQuestion`

## Checklist

Before reporting completion, confirm:

- [ ] `make fix` ran successfully at Step 0
- [ ] `ORIGINAL_HEAD` captured before any commit
- [ ] Commits were made on a non-protected branch
- [ ] Repository was not mid-merge / mid-rebase / mid-cherry-pick
- [ ] The user approved the proposed grouping (unless `--dry-run`)
- [ ] Lefthook skip notice was shown to the user with the dynamic command list
- [ ] Each commit has a single prefix
- [ ] Each commit message is in Japanese and includes the `Co-Authored-By` footer
- [ ] Each commit used `--no-verify` and was passed via HEREDOC
- [ ] `git add` named files explicitly (no `-A` / `.`)
- [ ] Generated artifacts co-located with their source change
- [ ] User was reminded to run lefthook / `make lint` / `make test` before pushing
- [ ] No automatic push was performed
