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

## Scope and confirmation

- Execute the concrete operation the user requested. Do not ask for confirmation of a command
  that is a normal, non-destructive step of that requested operation.
- Do not broaden the task: configuration changes for Codex belong under `.codex/` or this file.
  Do not modify `AGENTS.md` for Codex-only behavior unless the user explicitly asks.
- Before changing behavior outside the user's stated scope, report the consequence and ask for
  direction.
