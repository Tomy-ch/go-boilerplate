# Codex CLI Operational Safeguards

This file applies only to Codex CLI. It supplements `AGENTS.md`; shared repository rules remain
authoritative unless this file states a Codex-specific execution detail.

## Git operations

- When synchronizing a feature branch with its PR base, run `git merge origin/<pr-base>` without
  supplying a commit message. Preserve Git's generated merge subject and body; do not rename it
  to a conventional-commit-style subject such as `Merge: ...`.
- Do not amend, rebase, squash, or force-push unless the user explicitly requests that operation.
- Treat a user instruction to resolve a remote conflict as authorization to fetch, merge the PR
  base, resolve the resulting conflicts, verify the result, commit the merge, and push the
  feature branch. Do not stop to request confirmation for the already-requested `git push`.
- For a verified, task-scoped change on an existing feature PR branch, stage the change, create
  the required commit, and perform an ordinary push without asking again. Inspect `git status`
  and the diff first to exclude unrelated work; use `git add .` when the worktree contains only
  the task's changes, otherwise stage the task-scoped paths explicitly.

## Scope and confirmation

- A user request to implement, change, fix, or configure this repository authorizes every
  non-destructive edit required anywhere inside the repository. Do not ask whether an individual
  file may be changed.
- Do not ask for confirmation of normal implementation commands, including `git add`, `git
  commit`, and an ordinary `git push` to the current feature branch. Check the worktree to avoid
  including unrelated changes, but perform that check without pausing for confirmation.
- Do not broaden the task: configuration changes for Codex belong under `.codex/` or this file.
  Do not modify `AGENTS.md` for Codex-only behavior unless the user explicitly asks.
- Before changing behavior outside the user's stated scope, report the consequence and ask for
  direction.
