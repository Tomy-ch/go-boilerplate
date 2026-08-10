# Codex CLI Operational Safeguards

This file applies only to Codex CLI. It supplements `AGENTS.md`; shared repository rules remain
authoritative unless this file states a Codex-specific execution detail.

## Git operations

- When synchronizing a feature branch with its PR base, run `git merge origin/<pr-base>` without
  supplying a commit message. Preserve Git's generated merge subject and body; do not rename it
  to a conventional-commit-style subject such as `Merge: ...`.
- Do not amend, rebase, squash, or force-push unless the user explicitly requests that operation.
- Use this repository's `/commit` workflow for every commit. Do not stage, commit, or push
  manually; `/commit` owns change grouping, verification, and an ordinary push for an open PR.
- Use this repository's `/submit-pr` workflow to create or update a pull request. It owns the
  PR template, review decision, and required push confirmation.

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
